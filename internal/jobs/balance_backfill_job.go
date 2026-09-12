package jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type balanceUserSvc interface {
	GetAllActiveUserIDs(ctx context.Context) ([]int64, error)
}

type balanceAccountLister interface {
	ListOpenAccountIDs(ctx context.Context, afterID int64, limit int) ([]int64, error)
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
	w.logger.Info("Starting scheduled backfill fan-out...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Backfill fan-out failed", zap.Error(err))
		return err
	}
	w.logger.Info("Backfill fan-out completed successfully")
	return nil
}

type BalanceBackfillJob struct {
	logger             *zap.Logger
	userSvc            balanceUserSvc
	accountLister      balanceAccountLister
	dispatcher         jobqueue.Dispatcher
	userBatchSize      int
	reconcileBatchSize int
}

func NewBalanceBackfillJob(
	logger *zap.Logger,
	userSvc balanceUserSvc,
	accountLister balanceAccountLister,
	dispatcher jobqueue.Dispatcher,
	userBatchSize int,
	reconcileBatchSize int,
) *BalanceBackfillJob {
	if userBatchSize < 1 {
		userBatchSize = defaultUserBatchSize
	}
	if reconcileBatchSize < 1 {
		reconcileBatchSize = defaultReconcileBatchSize
	}
	return &BalanceBackfillJob{
		logger:             logger,
		userSvc:            userSvc,
		accountLister:      accountLister,
		dispatcher:         dispatcher,
		userBatchSize:      userBatchSize,
		reconcileBatchSize: reconcileBatchSize,
	}
}

const (
	defaultUserBatchSize      = 100
	defaultReconcileBatchSize = 500
)

func (j *BalanceBackfillJob) Run(ctx context.Context) error {
	reconcileBatches, err := j.fanOutReconcile(ctx)
	if err != nil {
		return err
	}

	backfillBatches, err := j.fanOutBackfill(ctx)
	if err != nil {
		return err
	}

	j.logger.Info("Balance fan-out enqueued",
		zap.Int("reconcile_batches", reconcileBatches),
		zap.Int("backfill_batches", backfillBatches))

	return nil
}

func (j *BalanceBackfillJob) fanOutBackfill(ctx context.Context) (int, error) {
	userIDs, err := j.userSvc.GetAllActiveUserIDs(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get user IDs: %w", err)
	}

	batches := 0
	for start := 0; start < len(userIDs); start += j.userBatchSize {
		end := min(start+j.userBatchSize, len(userIDs))

		args := jobqueue.BalanceBackfillBatchArgs{UserIDs: userIDs[start:end]}
		if err := j.dispatcher.Dispatch(ctx, args); err != nil {
			return batches, fmt.Errorf("enqueue backfill batch: %w", err)
		}
		batches++
	}

	return batches, nil
}

func (j *BalanceBackfillJob) fanOutReconcile(ctx context.Context) (int, error) {
	batches := 0

	for afterID := int64(0); ; {
		ids, err := j.accountLister.ListOpenAccountIDs(ctx, afterID, j.reconcileBatchSize)
		if err != nil {
			return batches, fmt.Errorf("list open accounts: %w", err)
		}
		if len(ids) == 0 {
			return batches, nil
		}
		afterID = ids[len(ids)-1]

		args := jobqueue.BalanceReconcileBatchArgs{AccountIDs: ids}
		if err := j.dispatcher.Dispatch(ctx, args); err != nil {
			return batches, fmt.Errorf("enqueue reconcile batch: %w", err)
		}
		batches++
	}
}
