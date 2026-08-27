package jobs

import (
	"context"
	"github.com/riverqueue/river"
	"sync"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AssetPriceHistoryBackfillWorker struct {
	river.WorkerDefaults[jobqueue.AssetPriceHistoryBackfillArgs]
	logger *zap.Logger
	job    *AssetPriceHistoryBackfillJob
}

func NewAssetPriceHistoryBackfillWorker(logger *zap.Logger, job *AssetPriceHistoryBackfillJob) *AssetPriceHistoryBackfillWorker {
	return &AssetPriceHistoryBackfillWorker{logger: logger, job: job}
}

func (w *AssetPriceHistoryBackfillWorker) Timeout(*river.Job[jobqueue.AssetPriceHistoryBackfillArgs]) time.Duration {
	return 5 * time.Minute
}

func (w *AssetPriceHistoryBackfillWorker) Work(ctx context.Context, _ *river.Job[jobqueue.AssetPriceHistoryBackfillArgs]) error {
	w.logger.Info("Starting asset price history backfill ...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Price history backfill failed", zap.Error(err))
		return err
	}
	w.logger.Info("Price history backfill completed")
	return nil
}

type AssetPriceHistoryBackfillJob struct {
	logger            *zap.Logger
	investmentSvc     services.InvestmentServiceInterface
	concurrentWorkers int
}

func NewAssetPriceHistoryBackfillJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	concurrentWorkers int,
) *AssetPriceHistoryBackfillJob {
	if concurrentWorkers < 1 {
		concurrentWorkers = defaultWorkers
	}
	return &AssetPriceHistoryBackfillJob{
		logger:            logger,
		investmentSvc:     investmentSvc,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AssetPriceHistoryBackfillJob) Run(ctx context.Context) error {
	assets, err := j.investmentSvc.GetAssetsForPriceBackfill(ctx)
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		j.logger.Info("No assets to backfill price history for")
		return nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	var (
		mu     sync.Mutex
		failed int
	)

	g := new(errgroup.Group)
	g.SetLimit(j.concurrentWorkers)

	for _, asset := range assets {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			err := j.investmentSvc.BackfillAssetPriceHistory(
				ctx, asset.ID, asset.Ticker, asset.EarliestTrade, today)
			if err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				return err
			}
			return nil
		})
	}
	firstErr := g.Wait()

	j.logger.Info("Backfill completed",
		zap.Int("total", len(assets)),
		zap.Int("failed", failed))

	if firstErr != nil {
		j.logger.Error("Backfill had failures",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	return ctx.Err()
}
