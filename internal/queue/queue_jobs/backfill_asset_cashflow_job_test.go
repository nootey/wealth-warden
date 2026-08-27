package queue_jobs_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"wealth-warden/internal/queue/queue_jobs"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.uber.org/zap/zaptest"
)

func runBackfill(w *queue_jobs.BackfillAssetCashFlowsWorker, ctx context.Context) error {
	return w.Work(ctx, &river.Job[queue_jobs.BackfillAssetCashFlowsArgs]{
		JobRow: &rivertype.JobRow{},
	})
}

type mockInvestmentRebuildSvc struct {
	userIDs    []int64
	userIDsErr error
	rebuildErr map[int64]error

	mu      sync.Mutex
	rebuilt []int64
}

func (m *mockInvestmentRebuildSvc) GetUserIDsWithInvestments(_ context.Context) ([]int64, error) {
	return m.userIDs, m.userIDsErr
}

func (m *mockInvestmentRebuildSvc) RebuildInvestmentDerivedData(_ context.Context, userID int64) error {
	if err := m.rebuildErr[userID]; err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebuilt = append(m.rebuilt, userID)
	return nil
}

func (m *mockInvestmentRebuildSvc) rebuiltIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Sorted(slices.Values(m.rebuilt))
}

func TestBackfillAssetCashFlowsJob_NoUsers(t *testing.T) {
	worker := queue_jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(t),
		&mockInvestmentRebuildSvc{},
		2,
	)

	if err := runBackfill(worker, context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackfillAssetCashFlowsJob_GetUserIDsError(t *testing.T) {
	worker := queue_jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(t),
		&mockInvestmentRebuildSvc{userIDsErr: errors.New("db error")},
		2,
	)

	if err := runBackfill(worker, context.Background()); err == nil {
		t.Error("expected error when GetUserIDsWithInvestments fails")
	}
}

func TestBackfillAssetCashFlowsJob_Success(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{userIDs: []int64{1, 2, 3}}
	worker := queue_jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(t),
		invSvc,
		2,
	)

	if err := runBackfill(worker, context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(invSvc.rebuiltIDs()) != 3 {
		t.Errorf("expected 3 users rebuilt, got %d: %v", len(invSvc.rebuiltIDs()), invSvc.rebuiltIDs())
	}
}

func TestBackfillAssetCashFlowsJob_FailsRunButFinishesOtherUsers(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{
		userIDs:    []int64{1, 2, 3},
		rebuildErr: map[int64]error{2: errors.New("rebuild failed")},
	}
	worker := queue_jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(t),
		invSvc,
		2,
	)

	if err := runBackfill(worker, context.Background()); err == nil {
		t.Error("expected the run to fail so the queue retries it")
	}

	if len(invSvc.rebuiltIDs()) != 2 {
		t.Errorf("expected users 1 and 3 rebuilt, got %v", invSvc.rebuiltIDs())
	}
}

func TestBackfillAssetCashFlowsJob_StopsOnCancelledContext(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{userIDs: []int64{1, 2, 3, 4, 5}}
	worker := queue_jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(t),
		invSvc,
		2,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runBackfill(worker, ctx)
	if err == nil {
		t.Fatal("a cancelled run must not report success")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected the error to wrap context.Canceled, got: %v", err)
	}
	if got := invSvc.rebuiltIDs(); len(got) != 0 {
		t.Errorf("expected no users rebuilt after cancellation, got %v", got)
	}
}

// The single-worker queue replaced the advisory lock: it is the only thing now
// stopping two clear-then-rebuild runs from interleaving.
func TestRebuildJobsShareTheSingleWorkerQueue(t *testing.T) {
	args := []river.JobArgs{
		queue_jobs.BackfillAssetCashFlowsArgs{},
		queue_jobs.CorrectFeeAccountingArgs{},
		queue_jobs.MigrateZeroCostTradesArgs{},
	}

	for _, a := range args {
		withOpts, ok := a.(river.JobArgsWithInsertOpts)
		if !ok {
			t.Fatalf("%s does not pin a queue", a.Kind())
		}
		if got := withOpts.InsertOpts().Queue; got != queue_jobs.QueueRebuild {
			t.Errorf("%s runs on queue %q, want %q", a.Kind(), got, queue_jobs.QueueRebuild)
		}
	}
}
