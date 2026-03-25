package queue_test

import (
	"context"
	"errors"
	"testing"
	"wealth-warden/internal/queue"

	"go.uber.org/zap/zaptest"
)

type mockUserSvc struct {
	userIDs []int64
	err     error
}

func (m *mockUserSvc) GetAllActiveUserIDs(_ context.Context) ([]int64, error) {
	return m.userIDs, m.err
}

type mockAccountBackfillSvc struct {
	clearCashFlowsErr map[int64]error
	clearSnapshotsErr map[int64]error
	rebuildErr        map[int64]error
	clearedCashFlows  []int64
	clearedSnapshots  []int64
	rebuilt           []int64
}

func (m *mockAccountBackfillSvc) ClearInvestmentCashFlows(_ context.Context, userID int64) error {
	if err := m.clearCashFlowsErr[userID]; err != nil {
		return err
	}
	m.clearedCashFlows = append(m.clearedCashFlows, userID)
	return nil
}

func (m *mockAccountBackfillSvc) ClearInvestmentSnapshots(_ context.Context, userID int64) error {
	if err := m.clearSnapshotsErr[userID]; err != nil {
		return err
	}
	m.clearedSnapshots = append(m.clearedSnapshots, userID)
	return nil
}

func (m *mockAccountBackfillSvc) RebuildSnapshotsForUser(_ context.Context, userID int64) error {
	if err := m.rebuildErr[userID]; err != nil {
		return err
	}
	m.rebuilt = append(m.rebuilt, userID)
	return nil
}

type mockInvestmentBackfillSvc struct {
	backfillErr map[int64]error
	backfilled  []int64
}

func (m *mockInvestmentBackfillSvc) BackfillInvestmentCashFlows(_ context.Context, userID int64) error {
	if err := m.backfillErr[userID]; err != nil {
		return err
	}
	m.backfilled = append(m.backfilled, userID)
	return nil
}

func TestBackfillAssetCashFlowsJob_NoUsers(t *testing.T) {
	job := queue.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		&mockInvestmentBackfillSvc{},
		&mockAccountBackfillSvc{},
		&mockUserSvc{userIDs: []int64{}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackfillAssetCashFlowsJob_GetUserIDsError(t *testing.T) {
	job := queue.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		&mockInvestmentBackfillSvc{},
		&mockAccountBackfillSvc{},
		&mockUserSvc{err: errors.New("db error")},
	)

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error when GetAllActiveUserIDs fails")
	}
}

func TestBackfillAssetCashFlowsJob_Success(t *testing.T) {
	accSvc := &mockAccountBackfillSvc{}
	invSvc := &mockInvestmentBackfillSvc{}
	job := queue.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		invSvc,
		accSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(accSvc.rebuilt) != 3 {
		t.Errorf("expected 3 users rebuilt, got %d", len(accSvc.rebuilt))
	}
	if len(invSvc.backfilled) != 3 {
		t.Errorf("expected 3 users backfilled, got %d", len(invSvc.backfilled))
	}
}

func TestBackfillAssetCashFlowsJob_ContinuesOnError(t *testing.T) {
	// user 2 fails at ClearInvestmentCashFlows — users 1 and 3 should still complete fully
	accSvc := &mockAccountBackfillSvc{
		clearCashFlowsErr: map[int64]error{2: errors.New("clear failed")},
	}
	invSvc := &mockInvestmentBackfillSvc{}
	job := queue.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		invSvc,
		accSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("job should not return error on partial failure, got: %v", err)
	}

	// users 1 and 3 should have been fully processed
	if len(accSvc.rebuilt) != 2 {
		t.Errorf("expected 2 users rebuilt (1 and 3), got %d: %v", len(accSvc.rebuilt), accSvc.rebuilt)
	}
	if len(invSvc.backfilled) != 2 {
		t.Errorf("expected 2 users backfilled (1 and 3), got %d: %v", len(invSvc.backfilled), invSvc.backfilled)
	}
}

func TestBackfillAssetCashFlowsJob_SkipsRemainingStepsOnUserError(t *testing.T) {
	// user 2 fails at BackfillInvestmentCashFlows — RebuildSnapshots should not be called for user 2
	accSvc := &mockAccountBackfillSvc{}
	invSvc := &mockInvestmentBackfillSvc{
		backfillErr: map[int64]error{2: errors.New("backfill failed")},
	}
	job := queue.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(t),
		invSvc,
		accSvc,
		&mockUserSvc{userIDs: []int64{1, 2, 3}},
	)

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("job should not return error on partial failure, got: %v", err)
	}

	// all 3 users should have had cash flows cleared (step before the failing one)
	if len(accSvc.clearedCashFlows) != 3 {
		t.Errorf("expected 3 users cleared, got %d", len(accSvc.clearedCashFlows))
	}
	// only users 1 and 3 should have been rebuilt
	if len(accSvc.rebuilt) != 2 {
		t.Errorf("expected 2 users rebuilt (1 and 3), got %d: %v", len(accSvc.rebuilt), accSvc.rebuilt)
	}
}
