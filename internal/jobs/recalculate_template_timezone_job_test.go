package jobs_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type mockTemplateRescheduler struct {
	templates       []models.TransactionTemplate
	getErr          error
	updateErr       error
	capturedUpdates []models.TemplateTimezoneUpdate
}

func (m *mockTemplateRescheduler) GetActiveTemplatesForUser(_ context.Context, _ int64) ([]models.TransactionTemplate, error) {
	return m.templates, m.getErr
}

func (m *mockTemplateRescheduler) BulkUpdateTemplateTimezone(_ context.Context, updates []models.TemplateTimezoneUpdate) error {
	m.capturedUpdates = updates
	return m.updateErr
}

func TestRecalculateTemplateTimezoneJob_InvalidTimezone(t *testing.T) {
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		&mockTemplateRescheduler{},
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Not/ATimezone"}

	err := worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args})
	assert.Error(t, err)
}

func TestRecalculateTemplateTimezoneJob_GetTemplatesError(t *testing.T) {
	repo := &mockTemplateRescheduler{getErr: errors.New("db error")}
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		repo,
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Europe/Ljubljana"}

	err := worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args})
	assert.Error(t, err)
}

func TestRecalculateTemplateTimezoneJob_NoTemplates(t *testing.T) {
	repo := &mockTemplateRescheduler{templates: []models.TransactionTemplate{}}
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		repo,
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Europe/Ljubljana"}

	err := worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args})
	require.NoError(t, err)
	assert.Nil(t, repo.capturedUpdates)
}

func TestRecalculateTemplateTimezoneJob_BulkUpdateError(t *testing.T) {
	repo := &mockTemplateRescheduler{
		templates: []models.TransactionTemplate{
			{ID: 1, NextRunAt: time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)},
		},
		updateErr: errors.New("update failed"),
	}
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		repo,
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Europe/Ljubljana"}

	err := worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args})
	assert.Error(t, err)
}

func TestRecalculateTemplateTimezoneJob_ReanchorsToSameLocalDate(t *testing.T) {
	// next_run_at is midnight UTC (00:00 UTC = 01:00 Ljubljana CET)
	// switching to Ljubljana should produce 2025-02-15 23:00 UTC (midnight Ljubljana)
	utcMidnight := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)

	repo := &mockTemplateRescheduler{
		templates: []models.TransactionTemplate{
			{ID: 1, NextRunAt: utcMidnight, DayOfMonth: 15},
		},
	}
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		repo,
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Europe/Ljubljana"}

	require.NoError(t, worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args}))
	require.Len(t, repo.capturedUpdates, 1)

	u := repo.capturedUpdates[0]
	assert.Equal(t, int64(1), u.ID)

	// Ljubljana is UTC+1 in winter, so midnight Ljubljana = 23:00 UTC previous day
	ljubljana, _ := time.LoadLocation("Europe/Ljubljana")
	local := u.NextRunAt.In(ljubljana)
	assert.Equal(t, 0, local.Hour())
	assert.Equal(t, 0, local.Minute())
	assert.Equal(t, 15, local.Day())
	assert.Equal(t, time.February, local.Month())
	assert.Equal(t, 15, u.DayOfMonth)
}

func TestRecalculateTemplateTimezoneJob_MultipleTemplates(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")

	// Midday UTC - local Paris date is the same as UTC date regardless of DST offset
	t1 := time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC)

	repo := &mockTemplateRescheduler{
		templates: []models.TransactionTemplate{
			{ID: 1, NextRunAt: t1, DayOfMonth: 10},
			{ID: 2, NextRunAt: t2, DayOfMonth: 20},
		},
	}
	worker := jobs.NewRecalculateTemplateTimezoneWorker(
		zaptest.NewLogger(t),
		repo,
	)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: "Europe/Paris"}

	require.NoError(t, worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args}))
	require.Len(t, repo.capturedUpdates, 2)

	for _, u := range repo.capturedUpdates {
		local := u.NextRunAt.In(paris)
		assert.Equal(t, 0, local.Hour(), "template %d: expected midnight Paris", u.ID)
		assert.Equal(t, 0, local.Minute())
	}

	assert.Equal(t, 10, repo.capturedUpdates[0].DayOfMonth)
	assert.Equal(t, 20, repo.capturedUpdates[1].DayOfMonth)
}
