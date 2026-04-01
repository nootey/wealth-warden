package queue

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"go.uber.org/zap"
)

type templateRescheduler interface {
	GetActiveTemplatesForUser(ctx context.Context, userID int64) ([]models.TransactionTemplate, error)
	BulkUpdateTemplateTimezone(ctx context.Context, updates []models.TemplateTimezoneUpdate) error
}

type RecalculateTemplateTimezoneJob struct {
	logger      *zap.Logger
	repo        templateRescheduler
	UserID      int64
	OldTimezone string
	NewTimezone string
}

func NewRecalculateTemplateTimezoneJob(
	logger *zap.Logger,
	repo templateRescheduler,
	userID int64,
	oldTimezone, newTimezone string,
) *RecalculateTemplateTimezoneJob {
	return &RecalculateTemplateTimezoneJob{
		logger:      logger,
		repo:        repo,
		UserID:      userID,
		OldTimezone: oldTimezone,
		NewTimezone: newTimezone,
	}
}

func (j *RecalculateTemplateTimezoneJob) Process(ctx context.Context) error {
	newLoc, err := time.LoadLocation(j.NewTimezone)
	if err != nil {
		j.logger.Error("Invalid new timezone", zap.String("timezone", j.NewTimezone), zap.Error(err))
		return err
	}

	templates, err := j.repo.GetActiveTemplatesForUser(ctx, j.UserID)
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
		y, m, d := t.NextRunAt.In(newLoc).Date()
		updates = append(updates, models.TemplateTimezoneUpdate{
			ID:         t.ID,
			NextRunAt:  time.Date(y, m, d, 0, 0, 0, 0, newLoc).UTC(),
			DayOfMonth: d,
		})
	}

	if err := j.repo.BulkUpdateTemplateTimezone(ctx, updates); err != nil {
		j.logger.Error("Failed to bulk update template timezones", zap.Int64("userID", j.UserID), zap.Error(err))
		return err
	}

	j.logger.Info("Recalculated template timezones",
		zap.Int64("userID", j.UserID),
		zap.String("oldTZ", j.OldTimezone),
		zap.String("newTZ", j.NewTimezone),
		zap.Int("count", len(updates)),
	)
	return nil
}
