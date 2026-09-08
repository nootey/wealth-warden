package jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type balanceAccountSvc interface {
	BackfillBalancesForUser(ctx context.Context, userID int64, from, to string) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64, from time.Time) error
}

type BalanceBackfillBatchWorker struct {
	river.WorkerDefaults[jobqueue.BalanceBackfillBatchArgs]
	logger     *zap.Logger
	accountSvc balanceAccountSvc
}

func NewBalanceBackfillBatchWorker(logger *zap.Logger, accountSvc balanceAccountSvc) *BalanceBackfillBatchWorker {
	return &BalanceBackfillBatchWorker{logger: logger, accountSvc: accountSvc}
}

func (w *BalanceBackfillBatchWorker) Timeout(*river.Job[jobqueue.BalanceBackfillBatchArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *BalanceBackfillBatchWorker) Work(ctx context.Context, job *river.Job[jobqueue.BalanceBackfillBatchArgs]) error {
	return w.Run(ctx, job.Args.UserIDs)
}

// One broken user makes its batch fail on every attempt, and each attempt redoes the
// other users. Redoing is safe, because every step rebuilds from transactions.
func (w *BalanceBackfillBatchWorker) Run(ctx context.Context, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}

	fromDate := time.Now().AddDate(0, 0, -1)
	to := time.Now().Format("2006-01-02")
	from := fromDate.Format("2006-01-02")

	failed := 0
	for _, userID := range userIDs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := w.backfillUser(ctx, userID, fromDate, from, to); err != nil {
			failed++
			w.logger.Error("Balance backfill failed for user",
				zap.Int64("userID", userID),
				zap.Error(err))
		}
	}

	if failed > 0 {
		return fmt.Errorf("balance backfill failed for %d of %d users", failed, len(userIDs))
	}
	return nil
}

// The market value update is scoped to investment and crypto accounts, so a user
// without them matches no rows.
func (w *BalanceBackfillBatchWorker) backfillUser(ctx context.Context, userID int64, fromDate time.Time, from, to string) error {
	if err := w.accountSvc.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
		return err
	}
	if err := w.accountSvc.UpdateSnapshotMarketValues(ctx, userID, fromDate); err != nil {
		return fmt.Errorf("market value update: %w", err)
	}
	return nil
}
