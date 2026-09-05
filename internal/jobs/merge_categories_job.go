package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type categoryMerger interface {
	MergeCategories(ctx context.Context, userID, sourceID, destinationID int64) (int64, error)
}

type MergeCategoriesWorker struct {
	river.WorkerDefaults[jobqueue.MergeCategoriesArgs]
	logger      *zap.Logger
	transaction categoryMerger
}

func NewMergeCategoriesWorker(logger *zap.Logger, transaction categoryMerger) *MergeCategoriesWorker {
	return &MergeCategoriesWorker{logger: logger, transaction: transaction}
}

func (w *MergeCategoriesWorker) Work(ctx context.Context, job *river.Job[jobqueue.MergeCategoriesArgs]) error {
	args := job.Args

	moved, err := w.transaction.MergeCategories(ctx, args.UserID, args.InternalSourceCategoryID, args.InternalDestinationCategoryID)
	if err != nil {
		w.logger.Error("Failed to merge categories",
			zap.Int64("userID", args.UserID),
			zap.Int64("sourceID", args.InternalSourceCategoryID),
			zap.Int64("destinationID", args.InternalDestinationCategoryID),
			zap.Error(err))
		return err
	}

	w.logger.Info("Merged categories",
		zap.Int64("userID", args.UserID),
		zap.String("source", args.SourceCategory),
		zap.String("destination", args.DestinationCategory),
		zap.Int64("transactions", moved),
	)

	return nil
}
