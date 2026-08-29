package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type balanceUserSvc interface {
	GetAllActiveUserIDs(ctx context.Context) ([]int64, error)
}

type balanceAccountSvc interface {
	BackfillBalancesForUser(ctx context.Context, userID int64, from, to string) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64, from time.Time) error
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

type BalanceBackfillJob struct {
	logger            *zap.Logger
	userSvc           balanceUserSvc
	accountSvc        balanceAccountSvc
	concurrentWorkers int
}

func NewBalanceBackfillJob(
	logger *zap.Logger,
	userSvc balanceUserSvc,
	accountSvc balanceAccountSvc,
	concurrentWorkers int,
) *BalanceBackfillJob {
	if concurrentWorkers < 1 {
		concurrentWorkers = defaultWorkers
	}
	return &BalanceBackfillJob{
		logger:            logger,
		userSvc:           userSvc,
		accountSvc:        accountSvc,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *BalanceBackfillJob) Run(ctx context.Context) error {

	userIDs, err := j.userSvc.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		j.logger.Info("No users to backfill")
		return nil
	}

	j.logger.Info("Backfilling balances", zap.Int("userCount", len(userIDs)))

	fromDate := time.Now().AddDate(0, 0, -1)
	to := time.Now().Format("2006-01-02")
	from := fromDate.Format("2006-01-02")

	var (
		mu     sync.Mutex
		failed int
	)

	g := new(errgroup.Group)
	g.SetLimit(j.concurrentWorkers)

	for _, userID := range userIDs {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			if err := j.backfillUser(ctx, userID, fromDate, from, to); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				return fmt.Errorf("user %d: %w", userID, err)
			}
			return nil
		})
	}
	firstErr := g.Wait()

	j.logger.Info("Balance backfill completed",
		zap.Int("total", len(userIDs)),
		zap.Int("failed", failed))

	if firstErr != nil {
		j.logger.Error("Balance backfill had failures",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	return ctx.Err()
}

// The market value update is scoped to investment and crypto accounts, so a user
// without them matches no rows.
func (j *BalanceBackfillJob) backfillUser(ctx context.Context, userID int64, fromDate time.Time, from, to string) error {
	if err := j.accountSvc.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
		return err
	}
	if err := j.accountSvc.UpdateSnapshotMarketValues(ctx, userID, fromDate); err != nil {
		return fmt.Errorf("market value update: %w", err)
	}
	return nil
}
