package jobs_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
)

type recordingBroadcaster struct {
	events map[int64][]ws.Event
}

func (b *recordingBroadcaster) Send(userID int64, event ws.Event) {
	if b.events == nil {
		b.events = make(map[int64][]ws.Event)
	}
	b.events[userID] = append(b.events[userID], event)
}

type mockCategoryReportSvc struct {
	fetchRows []models.CategoryReportDataRow
	fetchErr  error
	scope     *models.ReportAccountScope
	scopeErr  error

	processingErr error
	completedErr  error
	failedErr     error

	markCalls    int
	statuses     []string
	name         string
	filePath     string
	fileSize     int64
	failedReason string
}

func (m *mockCategoryReportSvc) MarkReportProcessing(_ context.Context, _ int64) error {
	m.markCalls++
	if m.processingErr != nil {
		return m.processingErr
	}
	m.statuses = append(m.statuses, "processing")
	return nil
}

func (m *mockCategoryReportSvc) MarkReportCompleted(_ context.Context, _ int64, name, filePath string, fileSize int64, _ time.Time) error {
	m.markCalls++
	if m.completedErr != nil {
		return m.completedErr
	}
	m.statuses = append(m.statuses, "completed")
	m.name, m.filePath, m.fileSize = name, filePath, fileSize
	return nil
}

func (m *mockCategoryReportSvc) MarkReportFailed(_ context.Context, _ int64, reason string) error {
	m.markCalls++
	if m.failedErr != nil {
		return m.failedErr
	}
	m.statuses = append(m.statuses, "failed")
	m.failedReason = reason
	return nil
}

func (m *mockCategoryReportSvc) FetchCategoryReportData(_ context.Context, _ int64, _ models.CategoryReportParams) ([]models.CategoryReportDataRow, error) {
	return m.fetchRows, m.fetchErr
}

func (m *mockCategoryReportSvc) FindReportAccountScope(_ context.Context, _, _ int64) (*models.ReportAccountScope, error) {
	if m.scopeErr != nil {
		return nil, m.scopeErr
	}
	if m.scope != nil {
		return m.scope, nil
	}
	return &models.ReportAccountScope{Name: "Main", Subtype: "checking_account"}, nil
}

func (m *mockCategoryReportSvc) lastStatus() string {
	if len(m.statuses) == 0 {
		return ""
	}
	return m.statuses[len(m.statuses)-1]
}

var sampleRows = []models.CategoryReportDataRow{
	{Year: 2024, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5000)},
	{Year: 2024, Month: 2, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5200)},
	{Year: 2024, Month: 3, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5100)},
}

func runReportJob(t *testing.T, svc *mockCategoryReportSvc, b ws.Broadcaster, args jobqueue.GenerateCategoryReportArgs) error {
	t.Helper()
	worker := jobs.NewGenerateCategoryReportWorker(zaptest.NewLogger(t), svc, b)
	return worker.Work(context.Background(), &river.Job[jobqueue.GenerateCategoryReportArgs]{Args: args})
}

func yearParams() models.CategoryReportParams {
	return models.CategoryReportParams{InflowCategoryIDs: []int64{1}, Years: []int{2024}}
}

func TestMain(m *testing.M) {
	code := m.Run()
	// The report job writes xlsx files under storage/reports relative to this
	// package dir; drop the whole tree so no empty storage/ dir is left behind.
	_ = os.RemoveAll("storage")
	os.Exit(code)
}

func TestGenerateCategoryReportJob_HappyPath(t *testing.T) {
	svc := &mockCategoryReportSvc{fetchRows: sampleRows}
	args := jobqueue.GenerateCategoryReportArgs{ReportID: 1, UserID: 1, Params: yearParams()}

	if err := runReportJob(t, svc, ws.NoopBroadcaster{}, args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := svc.statuses; len(got) != 2 || got[0] != "processing" || got[1] != "completed" {
		t.Fatalf("statuses = %v, want [processing completed]", got)
	}
	if svc.filePath == "" {
		t.Error("expected file_path to be set on completion")
	}
	if svc.fileSize <= 0 {
		t.Errorf("file_size = %d, want > 0", svc.fileSize)
	}
	if svc.name == "" {
		t.Error("expected a report name on completion")
	}
}

// A permanent failure must not come back to River: a retry would replay the same
// bad input, flap the row back to "processing", and re-notify the user.
func TestGenerateCategoryReportJob_RecordedFailureIsNotRetried(t *testing.T) {
	cases := []struct {
		name       string
		svc        *mockCategoryReportSvc
		wantReason string
	}{
		{"fetch error", &mockCategoryReportSvc{fetchErr: errors.New("db unavailable")}, "db unavailable"},
		{"no data", &mockCategoryReportSvc{fetchRows: nil}, jobs.ErrNoCategoryData.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := jobqueue.GenerateCategoryReportArgs{ReportID: 7, UserID: 3, Params: yearParams()}
			if err := runReportJob(t, tc.svc, ws.NoopBroadcaster{}, args); err != nil {
				t.Fatalf("error = %v, want nil so River does not retry", err)
			}
			if got := tc.svc.lastStatus(); got != "failed" {
				t.Errorf("last status = %q, want \"failed\"", got)
			}
			if tc.svc.failedReason != tc.wantReason {
				t.Errorf("reason = %q, want %q", tc.svc.failedReason, tc.wantReason)
			}
			for _, s := range tc.svc.statuses {
				if s == "completed" {
					t.Error("report was marked completed despite failing")
				}
			}
			if tc.svc.filePath != "" || tc.svc.name != "" {
				t.Error("completion fields were written despite failing")
			}
		})
	}
}

// The one failure worth retrying: the run ended without the row being marked, so
// it would otherwise sit in "processing" forever.
func TestGenerateCategoryReportJob_UnrecordedFailureIsRetried(t *testing.T) {
	svc := &mockCategoryReportSvc{
		fetchErr:  errors.New("db unavailable"),
		failedErr: errors.New("write failed"),
	}
	broadcaster := &recordingBroadcaster{}
	args := jobqueue.GenerateCategoryReportArgs{ReportID: 7, UserID: 3, Params: yearParams()}

	if err := runReportJob(t, svc, broadcaster, args); err == nil {
		t.Fatal("expected an error when the failure could not be recorded")
	}
	if len(broadcaster.events[3]) != 0 {
		t.Errorf("events = %#v, want none when the failure was not recorded", broadcaster.events[3])
	}
}

func TestGenerateCategoryReportJob_UnknownAccount_Fails(t *testing.T) {
	accountID := int64(99)
	svc := &mockCategoryReportSvc{fetchRows: sampleRows, scopeErr: gorm.ErrRecordNotFound}
	params := yearParams()
	params.AccountID = &accountID
	args := jobqueue.GenerateCategoryReportArgs{ReportID: 1, UserID: 1, Params: params}

	if err := runReportJob(t, svc, ws.NoopBroadcaster{}, args); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if got := svc.lastStatus(); got != "failed" {
		t.Errorf("last status = %q, want \"failed\"", got)
	}
}

func TestGenerateCategoryReportJob_InitialUpdateError_ReturnsImmediately(t *testing.T) {
	svc := &mockCategoryReportSvc{processingErr: errors.New("write failed")}
	args := jobqueue.GenerateCategoryReportArgs{ReportID: 1, UserID: 1, Params: yearParams()}

	if err := runReportJob(t, svc, ws.NoopBroadcaster{}, args); err == nil {
		t.Error("expected error when the initial status write fails")
	}
	if svc.markCalls != 1 {
		t.Errorf("mark calls = %d, want 1", svc.markCalls)
	}
}

func TestGenerateCategoryReportJob_BroadcastsOutcome(t *testing.T) {
	cases := []struct {
		name      string
		svc       *mockCategoryReportSvc
		wantEvent string
	}{
		{"completed", &mockCategoryReportSvc{fetchRows: sampleRows}, ws.TypeReportCompleted},
		{"failed", &mockCategoryReportSvc{fetchErr: errors.New("db unavailable")}, ws.TypeReportFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			broadcaster := &recordingBroadcaster{}
			args := jobqueue.GenerateCategoryReportArgs{ReportID: 9, UserID: 3, Params: yearParams()}

			_ = runReportJob(t, tc.svc, broadcaster, args)

			events := broadcaster.events[3]
			if len(events) != 1 {
				t.Fatalf("events for user 3 = %d, want 1", len(events))
			}
			if events[0].Type != tc.wantEvent {
				t.Fatalf("event type = %q, want %q", events[0].Type, tc.wantEvent)
			}
			if payload, ok := events[0].Payload.(ws.ReportPayload); !ok || payload.ReportID != 9 {
				t.Fatalf("payload = %#v, want ReportPayload{ReportID: 9}", events[0].Payload)
			}
		})
	}
}

func TestGenerateCategoryReportJob_AllTime_MultipleYears(t *testing.T) {
	svc := &mockCategoryReportSvc{fetchRows: []models.CategoryReportDataRow{
		{Year: 2022, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(4000)},
		{Year: 2023, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(4500)},
		{Year: 2024, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5000)},
	}}
	params := models.CategoryReportParams{InflowCategoryIDs: []int64{1}, AllTime: true}
	args := jobqueue.GenerateCategoryReportArgs{ReportID: 2, UserID: 1, Params: params}

	if err := runReportJob(t, svc, ws.NoopBroadcaster{}, args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := svc.lastStatus(); got != "completed" {
		t.Errorf("last status = %q, want \"completed\"", got)
	}
}
