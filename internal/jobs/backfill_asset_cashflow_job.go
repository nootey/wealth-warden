package jobs

import (
	"context"
	"fmt"
	"sync"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type investmentRebuildSvc interface {
	RebuildInvestmentDerivedData(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type BackfillAssetCashFlowsWorker struct {
	river.WorkerDefaults[jobqueue.BackfillAssetCashFlowsArgs]
	logger            *zap.Logger
	workers           int
	investmentService investmentRebuildSvc
}

func NewBackfillAssetCashFlowsWorker(logger *zap.Logger, investmentService investmentRebuildSvc, workers int) *BackfillAssetCashFlowsWorker {
	if workers < 1 {
		workers = defaultWorkers
	}
	return &BackfillAssetCashFlowsWorker{
		logger:            logger,
		workers:           workers,
		investmentService: investmentService,
	}
}

func (w *BackfillAssetCashFlowsWorker) Work(ctx context.Context, _ *river.Job[jobqueue.BackfillAssetCashFlowsArgs]) error {
	userIDs, err := w.investmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		w.logger.Info("No users to process")
		return nil
	}

	w.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	var (
		mu      sync.Mutex
		rebuilt int
		failed  int
	)

	g := new(errgroup.Group)
	g.SetLimit(w.workers)

	for _, userID := range userIDs {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			err := w.investmentService.RebuildInvestmentDerivedData(ctx, userID)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				return fmt.Errorf("user %d: %w", userID, err)
			}
			rebuilt++
			return nil
		})
	}
	firstErr := g.Wait()

	w.logger.Info("Investment rebuild completed",
		zap.Int("rebuilt", rebuilt),
		zap.Int("failed", failed),
		zap.Int("users_total", len(userIDs)))

	if firstErr != nil {
		w.logger.Error("Failed to rebuild investment derived data",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("rebuild stopped after %d of %d users: %w", rebuilt+failed, len(userIDs), err)
	}
	if firstErr != nil {
		return fmt.Errorf("%d of %d users failed to rebuild, first: %w", failed, len(userIDs), firstErr)
	}
	return nil
}
