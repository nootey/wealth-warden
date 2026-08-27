package jobs_test

import (
	"context"
	"sync"
	"testing"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

type AutomateTemplateJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAutomateTemplateJobSuite(t *testing.T) {
	suite.Run(t, new(AutomateTemplateJobTestSuite))
}

// Test that automate template job runs
func (s *AutomateTemplateJobTestSuite) TestAutomateTemplateJob_Success() {
	logger := zaptest.NewLogger(s.T())
	job := jobs.NewAutomateTemplateJob(logger, s.TC.App.TransactionService, nil, 0)

	err := job.Run(s.Ctx)
	s.NoError(err)
}

type recordedNotification struct {
	userID    int64
	title     string
	message   string
	notifType models.NotificationType
}

type recordingDispatcher struct {
	mu     sync.Mutex
	sent   []recordedNotification
	ctxErr error
}

func (d *recordingDispatcher) Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sent = append(d.sent, recordedNotification{userID, title, message, notifType})
	d.ctxErr = ctx.Err()
	return nil
}

func (d *recordingDispatcher) ofType(notifType models.NotificationType) []recordedNotification {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []recordedNotification
	for _, n := range d.sent {
		if n.notifType == notifType {
			out = append(out, n)
		}
	}
	return out
}

// The template already committed, so a cancelled run must still notify the user.
func TestAutomateTemplateJob_NotifiesAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	income := "income"
	tmpl := &models.TransactionTemplate{
		ID:              1,
		UserID:          7,
		Name:            "Salary",
		TemplateType:    "transaction",
		TransactionType: &income,
	}

	svc := mocks.NewMockTransactionServiceInterface(t)
	svc.EXPECT().GetTemplatesReadyToRun(mock.Anything).Return([]*models.TransactionTemplate{tmpl}, nil)
	svc.EXPECT().ProcessTemplate(mock.Anything, tmpl).
		Run(func(context.Context, *models.TransactionTemplate) { cancel() }).
		Return(nil)

	disp := &recordingDispatcher{}
	job := jobs.NewAutomateTemplateJob(zaptest.NewLogger(t), svc, disp, 1)

	require.ErrorIs(t, job.Run(ctx), context.Canceled)
	require.Len(t, disp.sent, 1)
	require.NoError(t, disp.ctxErr)
}

// A cancelled run drops the remaining phase without producing results. The count
// must say so, or the summary numbers silently stop adding up.
func TestAutomateTemplateJob_CountsCancelledTemplates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	income, expense := "income", "expense"
	in := &models.TransactionTemplate{
		ID: 1, UserID: 7, Name: "Salary",
		TemplateType: "transaction", TransactionType: &income,
	}
	out := &models.TransactionTemplate{
		ID: 2, UserID: 7, Name: "Rent",
		TemplateType: "transaction", TransactionType: &expense,
	}

	svc := mocks.NewMockTransactionServiceInterface(t)
	svc.EXPECT().GetTemplatesReadyToRun(mock.Anything).
		Return([]*models.TransactionTemplate{in, out}, nil)
	svc.EXPECT().ProcessTemplate(mock.Anything, in).
		Run(func(context.Context, *models.TransactionTemplate) { cancel() }).
		Return(nil)

	core, logs := observer.New(zapcore.InfoLevel)
	job := jobs.NewAutomateTemplateJob(zap.New(core), svc, nil, 1)

	require.ErrorIs(t, job.Run(ctx), context.Canceled)

	entries := logs.FilterMessage("Template processing completed").All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	require.Equal(t, int64(1), fields["success"])
	require.Equal(t, int64(1), fields["cancelled"])
}
