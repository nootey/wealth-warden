package queue_jobs

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type feeAccountingCorrectionSvc interface {
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
}

type CorrectFeeAccountingJob struct {
	logger            *zap.Logger
	lock              JobLock
	InvestmentService feeAccountingCorrectionSvc `json:"-"`
	UserService       userBackfillSvc            `json:"-"`
}

func (j *CorrectFeeAccountingJob) Type() string { return TypeCorrectFeeAccounting }

func NewCorrectFeeAccountingJob(
	logger *zap.Logger,
	lock JobLock,
	investmentService feeAccountingCorrectionSvc,
	userService userBackfillSvc,
) *CorrectFeeAccountingJob {
	return &CorrectFeeAccountingJob{
		logger:            logger,
		lock:              lock,
		InvestmentService: investmentService,
		UserService:       userService,
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

	userIDs, err := j.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		j.logger.Info("No users to process")
		return nil
	}

	j.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	failCount := 0

	for _, userID := range userIDs {
		if err := j.InvestmentService.CorrectFeeAccountingAndRebuild(ctx, userID); err != nil {
			j.logger.Error("Failed to correct fee accounting", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		j.logger.Info("User correction complete", zap.Int64("userID", userID))
	}

	j.logger.Info("Completed", zap.Int("success", len(userIDs)-failCount), zap.Int("failed", failCount))

	if failCount > 0 {
		return fmt.Errorf("%d of %d users failed", failCount, len(userIDs))
	}
	return nil
}
