package scheduler_jobs

import (
	"context"
	"time"
	"wealth-warden/internal/joblog"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AssetPriceHistoryBackfillJob struct {
	logger            *zap.Logger
	investmentSvc     services.InvestmentServiceInterface
	db                *gorm.DB
	concurrentWorkers int
}

func NewAssetPriceHistoryBackfillJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	db *gorm.DB,
	concurrentWorkers int,
) *AssetPriceHistoryBackfillJob {
	return &AssetPriceHistoryBackfillJob{
		logger:            logger,
		investmentSvc:     investmentSvc,
		db:                db,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AssetPriceHistoryBackfillJob) Run(ctx context.Context) error {
	type assetRow struct {
		ID             int64
		Ticker         string
		InvestmentType models.InvestmentType
		Currency       string
		EarliestTrade  time.Time
	}

	var assets []assetRow
	err := j.db.WithContext(ctx).Raw(`
		SELECT
			ia.id,
			ia.ticker,
			ia.investment_type,
			ia.currency,
			MIN(it.txn_date) AS earliest_trade
		FROM investment_assets ia
		JOIN investment_trades it ON it.asset_id = ia.id
		JOIN accounts a ON a.id = ia.account_id
		WHERE a.is_active = TRUE AND a.closed_at IS NULL
		GROUP BY ia.id, ia.ticker, ia.investment_type, ia.currency
	`).Scan(&assets).Error
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		j.logger.Info("No assets to backfill price history for")
		return nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	type result struct {
		assetID int64
		err     error
	}

	results := make(chan result, len(assets))
	processed := joblog.RunPool(ctx, assets, j.concurrentWorkers, func(ctx context.Context, asset assetRow) {
		err := j.investmentSvc.BackfillAssetPriceHistory(ctx, asset.ID, asset.Ticker, asset.InvestmentType, asset.EarliestTrade, today)
		results <- result{assetID: asset.ID, err: err}
	})
	close(results)

	failures := joblog.NewErrorGroup("asset_id")
	for r := range results {
		failures.Add(r.assetID, r.err)
	}

	failures.Log(j.logger, "Failed to backfill asset price history")

	j.logger.Info("Price history backfill completed",
		zap.Int("total", len(assets)),
		zap.Int("success", processed-failures.Count()),
		zap.Int("failed", failures.Count()),
		zap.Int("not_run", len(assets)-processed),
	)

	if processed < len(assets) {
		return joblog.StoppedEarly(ctx, processed, len(assets), "assets")
	}
	return nil
}

type AssetPriceHistoryBackfillArgs struct{}

func (AssetPriceHistoryBackfillArgs) Kind() string { return TypeAssetHistoryBackfill }

type AssetPriceHistoryBackfillWorker struct {
	river.WorkerDefaults[AssetPriceHistoryBackfillArgs]
	logger *zap.Logger
	job    *AssetPriceHistoryBackfillJob
}

func NewAssetPriceHistoryBackfillWorker(logger *zap.Logger, job *AssetPriceHistoryBackfillJob) *AssetPriceHistoryBackfillWorker {
	return &AssetPriceHistoryBackfillWorker{logger: logger, job: job}
}

func (w *AssetPriceHistoryBackfillWorker) Timeout(*river.Job[AssetPriceHistoryBackfillArgs]) time.Duration {
	return 5 * time.Minute
}

func (w *AssetPriceHistoryBackfillWorker) Work(ctx context.Context, _ *river.Job[AssetPriceHistoryBackfillArgs]) error {
	w.logger.Info("Starting asset price history backfill ...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Price history backfill failed", zap.Error(err))
		return err
	}
	w.logger.Info("Price history backfill completed")
	return nil
}
