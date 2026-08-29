package jobs_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type mockTemplateRescheduler struct {
	count   int
	err     error
	calls   int
	gotUser int64
	gotLoc  *time.Location
}

func (m *mockTemplateRescheduler) RecalculateTemplateTimezones(_ context.Context, userID int64, loc *time.Location) (int, error) {
	m.calls++
	m.gotUser, m.gotLoc = userID, loc
	return m.count, m.err
}

func runTimezoneJob(t *testing.T, svc *mockTemplateRescheduler, newTZ string) error {
	t.Helper()
	worker := jobs.NewRecalculateTemplateTimezoneWorker(zaptest.NewLogger(t), svc)
	args := jobqueue.RecalculateTemplateTimezoneArgs{UserID: 1, OldTimezone: "UTC", NewTimezone: newTZ}
	return worker.Work(context.Background(), &river.Job[jobqueue.RecalculateTemplateTimezoneArgs]{Args: args})
}

// A bad timezone string cannot be fixed by running again, so it must not come
// back to River. Settings only checks that the field is present.
func TestRecalculateTemplateTimezoneJob_InvalidTimezoneIsNotRetried(t *testing.T) {
	svc := &mockTemplateRescheduler{}

	require.NoError(t, runTimezoneJob(t, svc, "Not/ATimezone"))
	assert.Zero(t, svc.calls, "the service must not be called with an unusable timezone")
}

func TestRecalculateTemplateTimezoneJob_PassesLoadedLocation(t *testing.T) {
	svc := &mockTemplateRescheduler{count: 3}

	require.NoError(t, runTimezoneJob(t, svc, "Europe/Ljubljana"))
	assert.Equal(t, 1, svc.calls)
	assert.Equal(t, int64(1), svc.gotUser)
	require.NotNil(t, svc.gotLoc)
	assert.Equal(t, "Europe/Ljubljana", svc.gotLoc.String())
}

func TestRecalculateTemplateTimezoneJob_ServiceErrorIsRetried(t *testing.T) {
	svc := &mockTemplateRescheduler{err: errors.New("db error")}

	assert.Error(t, runTimezoneJob(t, svc, "Europe/Ljubljana"))
}
