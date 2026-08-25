package queue_jobs_test

import (
	"context"
	"errors"
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

type mockUserSvc struct {
	userIDs []int64
	err     error
}

func (m *mockUserSvc) GetAllActiveUserIDs(_ context.Context) ([]int64, error) {
	return m.userIDs, m.err
}

type mockInvestmentRebuildSvc struct {
	rebuildErr map[int64]error
	rebuilt    []int64
}

func (m *mockInvestmentRebuildSvc) RebuildInvestmentDerivedData(_ context.Context, userID int64) error {
	if err := m.rebuildErr[userID]; err != nil {
		return err
	}
	m.rebuilt = append(m.rebuilt, userID)
	return nil
}

func TestBackfillAssetCashFlowsJob_NoUsers(t *testing.T) {
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		&mockInvestmentRebuildSvc{},
		&mockUserSvc{userIDs: []int64{}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackfillAssetCashFlowsJob_GetUserIDsError(t *testing.T) {
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		&mockInvestmentRebuildSvc{},
		&mockUserSvc{err: errors.New("db error")},
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error when GetAllActiveUserIDs fails")
	}
}

func TestBackfillAssetCashFlowsJob_Success(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		invSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(invSvc.rebuilt) != 3 {
		t.Errorf("expected 3 users rebuilt, got %d: %v", len(invSvc.rebuilt), invSvc.rebuilt)
	}
}

func TestBackfillAssetCashFlowsJob_FailsRunButFinishesOtherUsers(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{
		rebuildErr: map[int64]error{2: errors.New("rebuild failed")},
	}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		grantingLock(),
		invSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected the run to fail so the queue retries it")
	}

	if len(invSvc.rebuilt) != 2 {
		t.Errorf("expected users 1 and 3 rebuilt, got %v", invSvc.rebuilt)
	}
}

func TestBackfillAssetCashFlowsJob_SkipsWhenLockHeld(t *testing.T) {
	invSvc := &mockInvestmentRebuildSvc{}
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		&stubLock{acquired: false},
		invSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("a held lock should not fail the job, got: %v", err)
	}

	if len(invSvc.rebuilt) != 0 {
		t.Errorf("expected no work when the lock is held, got rebuilt=%v", invSvc.rebuilt)
	}
}

func TestBackfillAssetCashFlowsJob_FailsWithoutLock(t *testing.T) {
	job := queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		nil,
		&mockInvestmentRebuildSvc{},
		&mockUserSvc{userIDs: []int64{1}},
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected an error when no lock is wired")
	}
}
