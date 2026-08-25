package queue_jobs_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"wealth-warden/internal/queue/queue_jobs"

	"go.uber.org/zap/zaptest"
)

type stubLock struct{ acquired bool }

func (s *stubLock) TryLock(_ context.Context, _ int64) (func(), bool, error) {
	if !s.acquired {
		return nil, false, nil
	}
	return func() {}, true, nil
}

func grantingLock() *stubLock { return &stubLock{acquired: true} }

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
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		&mockInvestmentRebuildSvc{},
		2,
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackfillAssetCashFlowsJob_GetUserIDsError(t *testing.T) {
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		&mockInvestmentRebuildSvc{userIDsErr: errors.New("db error")},
		2,
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error when GetUserIDsWithInvestments fails")
	}
}

func TestBackfillAssetCashFlowsJob_Success(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{userIDs: []int64{1, 2, 3}}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		invSvc,
		2,
	)

	if err := job.Process(context.Background()); err != nil {
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
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		invSvc,
		2,
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected the run to fail so the queue retries it")
	}

	if len(invSvc.rebuiltIDs()) != 2 {
		t.Errorf("expected users 1 and 3 rebuilt, got %v", invSvc.rebuiltIDs())
	}
}

func TestBackfillAssetCashFlowsJob_SkipsWhenLockHeld(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{userIDs: []int64{1, 2, 3}}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		&stubLock{acquired: false},
		invSvc,
		2,
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("a held lock should not fail the job, got: %v", err)
	}

	if len(invSvc.rebuiltIDs()) != 0 {
		t.Errorf("expected no work when the lock is held, got rebuilt=%v", invSvc.rebuiltIDs())
	}
}

func TestBackfillAssetCashFlowsJob_FailsWithoutLock(t *testing.T) {
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		nil,
		&mockInvestmentRebuildSvc{userIDs: []int64{1}},
		2,
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected an error when no lock is wired")
	}
}

func TestBackfillAssetCashFlowsJob_StopsOnCancelledContext(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{userIDs: []int64{1, 2, 3, 4, 5}}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		invSvc,
		2,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := job.Process(ctx)
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
