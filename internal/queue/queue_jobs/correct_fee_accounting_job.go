package queue_jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/joblog"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type feeAccountingCorrectionSvc interface {
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type CorrectFeeAccountingArgs struct{}

func (CorrectFeeAccountingArgs) Kind() string { return TypeCorrectFeeAccounting }

func (CorrectFeeAccountingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

type CorrectFeeAccountingWorker struct {
	river.WorkerDefaults[CorrectFeeAccountingArgs]
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

func (w *CorrectFeeAccountingWorker) Work(ctx context.Context, _ *river.Job[CorrectFeeAccountingArgs]) error {
	userIDs, err := w.investmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		w.logger.Info("No users to process")
		return nil
	}

	w.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	res := joblog.RunPerUser(ctx, userIDs, w.workers, w.investmentService.CorrectFeeAccountingAndRebuild)
	res.Log(w.logger, "Failed to correct fee accounting", len(userIDs))

	if res.Processed < len(userIDs) {
		return joblog.StoppedEarly(ctx, res.Processed, len(userIDs), "users")
	}
	if failCount := res.Failures.Count(); failCount > 0 {
		return fmt.Errorf("%d of %d users failed", failCount, len(userIDs))
	}
	return nil
}
