package jobs

import (
	"context"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type templateRescheduler interface {
	GetActiveTemplatesForUser(ctx context.Context, userID int64) ([]models.TransactionTemplate, error)
	BulkUpdateTemplateTimezone(ctx context.Context, updates []models.TemplateTimezoneUpdate) error
}

type RecalculateTemplateTimezoneWorker struct {
	river.WorkerDefaults[jobqueue.RecalculateTemplateTimezoneArgs]
	logger *zap.Logger
	repo   templateRescheduler
}

func NewRecalculateTemplateTimezoneWorker(logger *zap.Logger, repo templateRescheduler) *RecalculateTemplateTimezoneWorker {
	return &RecalculateTemplateTimezoneWorker{logger: logger, repo: repo}
}

func (w *RecalculateTemplateTimezoneWorker) Work(ctx context.Context, job *river.Job[jobqueue.RecalculateTemplateTimezoneArgs]) error {
	args := job.Args

	newLoc, err := time.LoadLocation(args.NewTimezone)
	if err != nil {
		w.logger.Error("Invalid new timezone", zap.String("timezone", args.NewTimezone), zap.Error(err))
		return err
	}

	templates, err := w.repo.GetActiveTemplatesForUser(ctx, args.UserID)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		return nil
	}

	updates := make([]models.TemplateTimezoneUpdate, 0, len(templates))
	for _, t := range templates {
		// Re-anchor to the same local calendar date in the new timezone.
		// E.g. next_run_at 23:00 UTC (midnight Paris CET) becomes 05:00 UTC (midnight NYC EST).
		// DayOfMonth is preserved as-is: it is the intended anchor day (e.g. 31), which may
		// differ from the actual day in next_run_at when the month is shorter.
		y, m, d := t.NextRunAt.In(newLoc).Date()
		updates = append(updates, models.TemplateTimezoneUpdate{
			ID:         t.ID,
			NextRunAt:  time.Date(y, m, d, 0, 0, 0, 0, newLoc).UTC(),
			DayOfMonth: t.DayOfMonth,
		})
	}

	if err := w.repo.BulkUpdateTemplateTimezone(ctx, updates); err != nil {
		w.logger.Error("Failed to bulk update template timezones", zap.Int64("userID", args.UserID), zap.Error(err))
		return err
	}

	w.logger.Info("Recalculated template timezones",
		zap.Int64("userID", args.UserID),
		zap.String("oldTZ", args.OldTimezone),
		zap.String("newTZ", args.NewTimezone),
		zap.Int("count", len(updates)),
	)
	return nil
}
