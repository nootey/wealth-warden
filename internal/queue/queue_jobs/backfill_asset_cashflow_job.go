package queue_jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/joblog"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type investmentRebuildSvc interface {
	RebuildInvestmentDerivedData(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type BackfillAssetCashFlowsArgs struct{}

func (BackfillAssetCashFlowsArgs) Kind() string { return TypeBackfillAssetCashFlows }

func (BackfillAssetCashFlowsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

type BackfillAssetCashFlowsWorker struct {
	river.WorkerDefaults[BackfillAssetCashFlowsArgs]
	logger            *zap.Logger
	workers           int
	investmentService investmentRebuildSvc
}

func NewBackfillAssetCashFlowsWorker(logger *zap.Logger, investmentService investmentRebuildSvc, workers int) *BackfillAssetCashFlowsWorker {
	return &BackfillAssetCashFlowsWorker{
		logger:            logger,
		workers:           workers,
		investmentService: investmentService,
	}
}

func (w *BackfillAssetCashFlowsWorker) Work(ctx context.Context, _ *river.Job[BackfillAssetCashFlowsArgs]) error {
	userIDs, err := w.investmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		w.logger.Info("No users to process")
		return nil
	}

	w.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	res := runPerUser(ctx, userIDs, w.workers, w.investmentService.RebuildInvestmentDerivedData)
	res.log(w.logger, "Failed to rebuild investment derived data", len(userIDs))

	if res.processed < len(userIDs) {
		return joblog.StoppedEarly(ctx, res.processed, len(userIDs), "users")
	}
	if failCount := res.failures.Count(); failCount > 0 {
		return fmt.Errorf("%d of %d users failed", failCount, len(userIDs))
	}
	return nil
}
