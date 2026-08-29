package jobs_test

import (
	"context"
	"errors"
	"testing"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRecurringTransactions_FundsGoalsAfterTemplates(t *testing.T) {
	var order []string

	worker := jobs.NewRecurringTransactionsWorker(zap.NewNop(),
		func(context.Context) error { order = append(order, "templates"); return nil },
		func(context.Context) error { order = append(order, "goals"); return nil })

	require.NoError(t, worker.Work(context.Background(), &river.Job[jobqueue.RecurringTransactionsArgs]{}))
	require.Equal(t, []string{"templates", "goals"}, order)
}

// Balances would be stale, so River retries instead. A missed day is caught up
// by a later run.
func TestRecurringTransactions_SkipsGoalsWhenTemplatesFail(t *testing.T) {
	tmplErr := errors.New("failed to get templates")
	funded := false

	worker := jobs.NewRecurringTransactionsWorker(zap.NewNop(),
		func(context.Context) error { return tmplErr },
		func(context.Context) error { funded = true; return nil })

	err := worker.Work(context.Background(), &river.Job[jobqueue.RecurringTransactionsArgs]{})
	require.False(t, funded)
	require.ErrorIs(t, err, tmplErr)
}

func TestRecurringTransactions_SkipsGoalsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	funded := false

	worker := jobs.NewRecurringTransactionsWorker(zap.NewNop(),
		func(context.Context) error { cancel(); return nil },
		func(context.Context) error { funded = true; return nil })

	err := worker.Work(ctx, &river.Job[jobqueue.RecurringTransactionsArgs]{})
	require.False(t, funded)
	require.ErrorIs(t, err, context.Canceled)
}
