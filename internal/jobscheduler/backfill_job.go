package jobscheduler

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"

	"go.uber.org/zap"
)

type BackfillJob struct {
	logger    *zap.Logger
	container *bootstrap.Container
}

func NewBackfillJob(logger *zap.Logger, container *bootstrap.Container) *BackfillJob {
	return &BackfillJob{
		logger:    logger,
		container: container,
	}
}

func (j *BackfillJob) Run(ctx context.Context) error {

	userIDs, err := j.container.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		j.logger.Info("No users to backfill")
		return nil
	}

	j.logger.Info("Backfilling balances", zap.Int("userCount", len(userIDs)))

	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	successCount := 0
	failCount := 0

	for _, userID := range userIDs {
		if err := j.container.AccountService.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
			j.logger.Error("Backfill failed for user",
				zap.Int64("userID", userID),
				zap.Error(err))
			failCount++
		} else {
			successCount++
		}
	}

	j.logger.Info("Backfill completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	return nil
}
