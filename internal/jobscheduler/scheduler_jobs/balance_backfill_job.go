package scheduler_jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/joblog"

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
		concurrentWorkers: workerCount(concurrentWorkers),
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

	investmentUserIDs, err := j.container.InvestmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users with investments: %w", err)
	}
	hasInvestments := make(map[int64]bool, len(investmentUserIDs))
	for _, uid := range investmentUserIDs {
		hasInvestments[uid] = true
	}

	j.logger.Info("Backfilling balances", zap.Int("userCount", len(userIDs)))

	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	type result struct {
		userID int64
		err    error
		mvErr  error
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
				r := result{userID: uid}
				r.err = j.container.AccountService.BackfillBalancesForUser(ctx, uid, from, to)
				if r.err == nil && hasInvestments[uid] {
					r.mvErr = j.container.AccountService.UpdateSnapshotMarketValues(ctx, uid)
				}
				results <- r
			}
		}()
	}

	for _, uid := range userIDs {
		jobs <- uid
	}
	close(jobs)

	wg.Wait()
	close(results)

	failures := joblog.NewErrorGroup("userID")
	mvFailures := joblog.NewErrorGroup("userID")
	processed := 0
	for r := range results {
		processed++
		failures.Add(r.userID, r.err)
		mvFailures.Add(r.userID, r.mvErr)
	}

	failures.Log(j.logger, "Backfill failed for user")
	mvFailures.Log(j.logger, "Failed to update snapshot market values after backfill")

	failCount := failures.Count()
	j.logger.Info("Backfill completed",
		zap.Int("total", len(userIDs)),
		zap.Int("success", processed-failCount),
		zap.Int("failed", failCount),
		zap.Int("not_run", len(userIDs)-processed))

	if processed < len(userIDs) {
		return joblog.StoppedEarly(ctx, processed, len(userIDs), "users")
	}
	if failCount > 0 || mvFailures.Count() > 0 {
		return fmt.Errorf("%d of %d users failed to backfill, %d failed the market value update",
			failCount, len(userIDs), mvFailures.Count())
	}
	return nil
}
