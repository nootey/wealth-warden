package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type accountMerger interface {
	MergeAccount(ctx context.Context, userID, sourceID, destinationID int64) error
}

type MergeAccountsWorker struct {
	river.WorkerDefaults[jobqueue.MergeAccountsArgs]
	logger  *zap.Logger
	account accountMerger
}

func NewMergeAccountsWorker(logger *zap.Logger, account accountMerger) *MergeAccountsWorker {
	return &MergeAccountsWorker{logger: logger, account: account}
}

func (w *MergeAccountsWorker) Work(ctx context.Context, job *river.Job[jobqueue.MergeAccountsArgs]) error {
	args := job.Args

	if err := w.account.MergeAccount(ctx, args.UserID, args.InternalSourceAccountID, args.InternalDestinationAccountID); err != nil {
		w.logger.Error("Failed to merge accounts",
			zap.Int64("userID", args.UserID),
			zap.Int64("sourceID", args.InternalSourceAccountID),
			zap.Int64("destinationID", args.InternalDestinationAccountID),
			zap.Error(err))
		return err
	}

	w.logger.Info("Merged accounts",
		zap.Int64("userID", args.UserID),
		zap.String("source", args.SourceAccount),
		zap.String("destination", args.DestinationAccount),
	)

	return nil
}
