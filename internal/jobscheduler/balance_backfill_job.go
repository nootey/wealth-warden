package jobscheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"

	"go.uber.org/zap"
)

type BalanceBackfillJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	concurrentWorkers int
}

func NewBalanceBackfillJob(logger *zap.Logger, container *bootstrap.ServiceContainer, concurrentWorkers int) *BalanceBackfillJob {
	return &BalanceBackfillJob{
		logger:            logger,
		container:         container,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *BalanceBackfillJob) Run(ctx context.Context) error {

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

	type result struct {
		userID int64
		err    error
	}

	jobs := make(chan int64, len(userIDs))
	results := make(chan result, len(userIDs))

	var wg sync.WaitGroup
	for i := 0; i < j.concurrentWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uid := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				err := j.container.AccountService.BackfillBalancesForUser(ctx, uid, from, to)
				results <- result{userID: uid, err: err}
			}
		}()
	}

	for _, uid := range userIDs {
		jobs <- uid
	}
	close(jobs)

	wg.Wait()
	close(results)

	successCount := 0
	failCount := 0
	for r := range results {
		if r.err != nil {
			j.logger.Error("Backfill failed for user",
				zap.Int64("userID", r.userID),
				zap.Error(r.err))
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
