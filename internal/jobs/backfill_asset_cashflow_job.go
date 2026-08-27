package jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
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

	res := runPool(ctx, userIDs, w.workers, "users", w.investmentService.RebuildInvestmentDerivedData)
	res.log(w.logger, "Failed to rebuild investment derived data")

	return res.err(ctx)
}
