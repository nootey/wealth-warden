package scheduler_jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/joblog"

	"github.com/riverqueue/river"
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

	results := make(chan result, len(userIDs))
	processed := joblog.RunPool(ctx, userIDs, j.concurrentWorkers, func(ctx context.Context, uid int64) {
		r := result{userID: uid}
		r.err = j.container.AccountService.BackfillBalancesForUser(ctx, uid, from, to)
		if r.err == nil && hasInvestments[uid] {
			r.mvErr = j.container.AccountService.UpdateSnapshotMarketValues(ctx, uid)
		}
		results <- r
	})
	close(results)

	failures := joblog.NewErrorGroup("userID")
	mvFailures := joblog.NewErrorGroup("userID")
	for r := range results {
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

type BalanceBackfillArgs struct{}

func (BalanceBackfillArgs) Kind() string { return TypeBalanceBackfill }

type BalanceBackfillWorker struct {
	river.WorkerDefaults[BalanceBackfillArgs]
	logger *zap.Logger
	job    *BalanceBackfillJob
}

func NewBalanceBackfillWorker(logger *zap.Logger, job *BalanceBackfillJob) *BalanceBackfillWorker {
	return &BalanceBackfillWorker{logger: logger, job: job}
}

func (w *BalanceBackfillWorker) Timeout(*river.Job[BalanceBackfillArgs]) time.Duration {
	return 3 * time.Minute
}

func (w *BalanceBackfillWorker) Work(ctx context.Context, _ *river.Job[BalanceBackfillArgs]) error {
	w.logger.Info("Starting scheduled backfill job...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Backfill failed", zap.Error(err))
		return err
	}
	w.logger.Info("Backfill completed successfully")
	return nil
}
