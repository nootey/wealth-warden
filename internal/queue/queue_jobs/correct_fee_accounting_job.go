package queue_jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/joblog"

	"go.uber.org/zap"
)

type feeAccountingCorrectionSvc interface {
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type CorrectFeeAccountingJob struct {
	logger            *zap.Logger
	lock              JobLock
	workers           int
	InvestmentService feeAccountingCorrectionSvc `json:"-"`
}

func (j *CorrectFeeAccountingJob) Type() string { return TypeCorrectFeeAccounting }

func NewCorrectFeeAccountingJob(
	logger *zap.Logger,
	lock JobLock,
	investmentService feeAccountingCorrectionSvc,
	workers int,
) *CorrectFeeAccountingJob {
	return &CorrectFeeAccountingJob{
		logger:            logger,
		lock:              lock,
		workers:           workers,
		InvestmentService: investmentService,
	}
}

func (j *CorrectFeeAccountingJob) Process(ctx context.Context) error {
	if j.lock == nil {
		return fmt.Errorf("correct fee accounting: no job lock wired")
	}

	release, acquired, err := j.lock.TryLock(ctx, LockKeyInvestmentRebuild)
	if err != nil {
		return fmt.Errorf("failed to take investment rebuild lock: %w", err)
	}
	if !acquired {
		j.logger.Info("Another investment rebuild holds the lock, skipping this run")
		return nil
	}
	defer release()

	userIDs, err := j.InvestmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		j.logger.Info("No users to process")
		return nil
	}

	j.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	res := runPerUser(ctx, userIDs, j.workers, j.InvestmentService.CorrectFeeAccountingAndRebuild)
	res.log(j.logger, "Failed to correct fee accounting", len(userIDs))

	if res.processed < len(userIDs) {
		return joblog.StoppedEarly(ctx, res.processed, len(userIDs), "users")
	}
	if failCount := res.failures.Count(); failCount > 0 {
		return fmt.Errorf("%d of %d users failed", failCount, len(userIDs))
	}
	return nil
}
