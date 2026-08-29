package jobs

import (
	"context"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type templateRescheduler interface {
	RecalculateTemplateTimezones(ctx context.Context, userID int64, loc *time.Location) (int, error)
}

type RecalculateTemplateTimezoneWorker struct {
	river.WorkerDefaults[jobqueue.RecalculateTemplateTimezoneArgs]
	logger      *zap.Logger
	transaction templateRescheduler
}

func NewRecalculateTemplateTimezoneWorker(logger *zap.Logger, transaction templateRescheduler) *RecalculateTemplateTimezoneWorker {
	return &RecalculateTemplateTimezoneWorker{logger: logger, transaction: transaction}
}

func (w *RecalculateTemplateTimezoneWorker) Work(ctx context.Context, job *river.Job[jobqueue.RecalculateTemplateTimezoneArgs]) error {
	args := job.Args

	// A bad timezone is permanent, so a retry cannot fix it. Settings only checks
	// that the field is present, which is how one reaches us in the first place.
	newLoc, err := time.LoadLocation(args.NewTimezone)
	if err != nil {
		w.logger.Error("Invalid new timezone, skipping template recalculation",
			zap.Int64("userID", args.UserID), zap.String("timezone", args.NewTimezone), zap.Error(err))
		return nil
	}

	count, err := w.transaction.RecalculateTemplateTimezones(ctx, args.UserID, newLoc)
	if err != nil {
		w.logger.Error("Failed to recalculate template timezones", zap.Int64("userID", args.UserID), zap.Error(err))
		return err
	}

	w.logger.Info("Recalculated template timezones",
		zap.Int64("userID", args.UserID),
		zap.String("oldTZ", args.OldTimezone),
		zap.String("newTZ", args.NewTimezone),
		zap.Int("count", count),
	)
	return nil
}
