package jobs

import (
	"context"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type balanceReconcileSvc interface {
	ReconcileAccounts(ctx context.Context, accountIDs []int64) ([]models.BalanceDrift, error)
}

type BalanceReconcileBatchWorker struct {
	river.WorkerDefaults[jobqueue.BalanceReconcileBatchArgs]
	logger     *zap.Logger
	balanceSvc balanceReconcileSvc
}

func NewBalanceReconcileBatchWorker(logger *zap.Logger, balanceSvc balanceReconcileSvc) *BalanceReconcileBatchWorker {
	return &BalanceReconcileBatchWorker{logger: logger, balanceSvc: balanceSvc}
}

func (w *BalanceReconcileBatchWorker) Timeout(*river.Job[jobqueue.BalanceReconcileBatchArgs]) time.Duration {
	return 5 * time.Minute
}

func (w *BalanceReconcileBatchWorker) Work(ctx context.Context, job *river.Job[jobqueue.BalanceReconcileBatchArgs]) error {
	return w.Run(ctx, job.Args.AccountIDs)
}

func (w *BalanceReconcileBatchWorker) Run(ctx context.Context, accountIDs []int64) error {
	drifted, err := w.balanceSvc.ReconcileAccounts(ctx, accountIDs)

	for _, d := range drifted {
		w.logger.Warn("Balance drifted from its transactions; repaired",
			zap.Int64("userID", d.UserID),
			zap.Int64("accountID", d.AccountID),
			zap.String("actual", d.Actual.String()),
			zap.String("expected", d.Expected.String()),
			zap.String("difference", d.Difference().String()))
	}

	return err
}
