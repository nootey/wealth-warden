package queue_jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/joblog"

	"go.uber.org/zap"
)

type investmentRebuildSvc interface {
	RebuildInvestmentDerivedData(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type BackfillAssetCashFlowsJob struct {
	logger            *zap.Logger
	lock              JobLock
	workers           int
	InvestmentService investmentRebuildSvc `json:"-"`
}

func (j *BackfillAssetCashFlowsJob) Type() string { return TypeBackfillAssetCashFlows }

func NewBackfillAssetCashFlowsJob(
	logger *zap.Logger,
	lock JobLock,
	investmentService investmentRebuildSvc,
	workers int,
) *BackfillAssetCashFlowsJob {
	return &BackfillAssetCashFlowsJob{
		logger:            logger,
		lock:              lock,
		workers:           workers,
		InvestmentService: investmentService,
	}
}

func (j *BackfillAssetCashFlowsJob) Process(ctx context.Context) error {
	if j.lock == nil {
		return fmt.Errorf("backfill asset cash flows: no job lock wired")
	}

	release, acquired, err := j.lock.TryLock(ctx, LockKeyInvestmentRebuild)
	if err != nil {
		return fmt.Errorf("failed to take investment rebuild lock: %w", err)
	}
	if !acquired {
		j.logger.Info("Another investment rebuild holds the lock, retrying later")
		return ErrRebuildLockHeld
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

	res := runPerUser(ctx, userIDs, j.workers, j.InvestmentService.RebuildInvestmentDerivedData)
	res.log(j.logger, "Failed to rebuild investment derived data", len(userIDs))

	if res.processed < len(userIDs) {
		return joblog.StoppedEarly(ctx, res.processed, len(userIDs), "users")
	}
	if failCount := res.failures.Count(); failCount > 0 {
		return fmt.Errorf("%d of %d users failed", failCount, len(userIDs))
	}
	return nil
}
