package jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type RecurringTransactionsWorker struct {
	river.WorkerDefaults[jobqueue.RecurringTransactionsArgs]
	logger    *zap.Logger
	templates func(context.Context) error
	goals     func(context.Context) error
}

func NewRecurringTransactionsWorker(logger *zap.Logger, templates, goals func(context.Context) error) *RecurringTransactionsWorker {
	return &RecurringTransactionsWorker{logger: logger, templates: templates, goals: goals}
}

func (w *RecurringTransactionsWorker) Timeout(*river.Job[jobqueue.RecurringTransactionsArgs]) time.Duration {
	return 16 * time.Minute
}

// Templates run first: auto-funding reads balances a template deposit may have
// just changed. Both steps skip work they already did today, so River retrying
// the whole job after a partial run is safe.
func (w *RecurringTransactionsWorker) Work(ctx context.Context, _ *river.Job[jobqueue.RecurringTransactionsArgs]) error {
	w.logger.Info("Starting recurring transactions job...")

	// Templates only error when the fetch itself failed or the run was
	// cancelled, so the balances funding reads would be wrong or stale. Let
	// River retry rather than fund against them; a missed day is caught up by
	// a later run, but a bad transfer is not.
	if err := w.templates(ctx); err != nil {
		w.logger.Error("Template processing failed, skipping auto-fund", zap.Error(err))
		return err
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("recurring transactions stopped early: %w", err)
	}

	if err := w.goals(ctx); err != nil {
		w.logger.Error("Savings goal auto-fund failed", zap.Error(err))
		return err
	}

	w.logger.Info("Recurring transactions completed successfully")
	return nil
}
