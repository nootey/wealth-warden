package jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type feeAccountingCorrectionSvc interface {
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type CorrectFeeAccountingWorker struct {
	river.WorkerDefaults[jobqueue.CorrectFeeAccountingArgs]
	logger            *zap.Logger
	workers           int
	investmentService feeAccountingCorrectionSvc
}

func NewCorrectFeeAccountingWorker(logger *zap.Logger, investmentService feeAccountingCorrectionSvc, workers int) *CorrectFeeAccountingWorker {
	return &CorrectFeeAccountingWorker{
		logger:            logger,
		workers:           workers,
		investmentService: investmentService,
	}
}

func (w *CorrectFeeAccountingWorker) Work(ctx context.Context, _ *river.Job[jobqueue.CorrectFeeAccountingArgs]) error {
	userIDs, err := w.investmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		w.logger.Info("No users to process")
		return nil
	}

	w.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	res := runPool(ctx, userIDs, w.workers, "users", w.investmentService.CorrectFeeAccountingAndRebuild)
	res.log(w.logger, "Failed to correct fee accounting")

	return res.err(ctx)
}
