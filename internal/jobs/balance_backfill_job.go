package jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobqueue"

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

	res := runPool(ctx, userIDs, j.concurrentWorkers, "users", func(ctx context.Context, uid int64) error {
		if err := j.container.AccountService.BackfillBalancesForUser(ctx, uid, from, to); err != nil {
			return err
		}
		if hasInvestments[uid] {
			if err := j.container.AccountService.UpdateSnapshotMarketValues(ctx, uid); err != nil {
				return fmt.Errorf("market value update: %w", err)
			}
		}
		return nil
	})
	res.log(j.logger, "Backfill failed for user")

	return res.err(ctx)
}

type BalanceBackfillWorker struct {
	river.WorkerDefaults[jobqueue.BalanceBackfillArgs]
	logger *zap.Logger
	job    *BalanceBackfillJob
}

func NewBalanceBackfillWorker(logger *zap.Logger, job *BalanceBackfillJob) *BalanceBackfillWorker {
	return &BalanceBackfillWorker{logger: logger, job: job}
}

func (w *BalanceBackfillWorker) Timeout(*river.Job[jobqueue.BalanceBackfillArgs]) time.Duration {
	return 3 * time.Minute
}

func (w *BalanceBackfillWorker) Work(ctx context.Context, _ *river.Job[jobqueue.BalanceBackfillArgs]) error {
	w.logger.Info("Starting scheduled backfill job...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Backfill failed", zap.Error(err))
		return err
	}
	w.logger.Info("Backfill completed successfully")
	return nil
}
