package queue_jobs_test

import (
	"context"
	"errors"
	"testing"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
	"go.uber.org/zap/zaptest"
)

type mockPnLSvc struct {
	recalculated []int64
	assetIDs     []int64
	recalcErr    error
	assetIDsErr  error
}

func (m *mockPnLSvc) RecalculateAssetPnL(_ context.Context, _ int64, assetID int64) error {
	if m.recalcErr != nil {
		return m.recalcErr
	}
	m.recalculated = append(m.recalculated, assetID)
	return nil
}

func (m *mockPnLSvc) GetAssetIDsForAccount(_ context.Context, _, _ int64) ([]int64, error) {
	return m.assetIDs, m.assetIDsErr
}

func (m *mockPnLSvc) UpdateSnapshotMarketValues(_ context.Context, _ int64) error {
	return nil
}

func ptr[T any](v T) *T { return &v }

func TestRecalculateAssetPnLJob_SingleAsset(t *testing.T) {
	svc := &mockPnLSvc{}
	broadcaster := &recordingBroadcaster{}
	worker := queue_jobs.NewRecalculateAssetPnLWorker(zaptest.NewLogger(t), svc, broadcaster)
	args := queue_jobs.RecalculateAssetPnLArgs{UserID: 1, AssetID: ptr(int64(42)), AccountID: nil}

	if err := worker.Work(context.Background(), &river.Job[queue_jobs.RecalculateAssetPnLArgs]{Args: args}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.recalculated) != 1 || svc.recalculated[0] != 42 {
		t.Errorf("expected asset 42 to be recalculated, got %v", svc.recalculated)
	}

	events := broadcaster.events[1]
	if len(events) != 1 || events[0].Type != ws.TypeAssetPnLSynced {
		t.Fatalf("expected one %s event, got %v", ws.TypeAssetPnLSynced, events)
	}
	payload, ok := events[0].Payload.(ws.AssetPnLPayload)
	if !ok || payload.AssetID == nil || *payload.AssetID != 42 {
		t.Errorf("expected payload with asset 42, got %#v", events[0].Payload)
	}
}

func TestRecalculateAssetPnLJob_AccountScope(t *testing.T) {
	svc := &mockPnLSvc{assetIDs: []int64{10, 20, 30}}
	broadcaster := &recordingBroadcaster{}
	worker := queue_jobs.NewRecalculateAssetPnLWorker(zaptest.NewLogger(t), svc, broadcaster)
	args := queue_jobs.RecalculateAssetPnLArgs{UserID: 1, AssetID: nil, AccountID: ptr(int64(5))}

	if err := worker.Work(context.Background(), &river.Job[queue_jobs.RecalculateAssetPnLArgs]{Args: args}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.recalculated) != 3 {
		t.Errorf("expected 3 assets recalculated, got %d", len(svc.recalculated))
	}
	for i, id := range []int64{10, 20, 30} {
		if svc.recalculated[i] != id {
			t.Errorf("expected asset %d at index %d, got %d", id, i, svc.recalculated[i])
		}
	}

	events := broadcaster.events[1]
	if len(events) != 1 {
		t.Fatalf("expected a single account-scoped event, got %v", events)
	}
	payload, ok := events[0].Payload.(ws.AssetPnLPayload)
	if !ok || payload.AccountID == nil || *payload.AccountID != 5 {
		t.Errorf("expected payload with account 5, got %#v", events[0].Payload)
	}
}

func TestRecalculateAssetPnLJob_NeitherAssetNorAccount(t *testing.T) {
	svc := &mockPnLSvc{}
	worker := queue_jobs.NewRecalculateAssetPnLWorker(zaptest.NewLogger(t), svc, ws.NoopBroadcaster{})
	args := queue_jobs.RecalculateAssetPnLArgs{UserID: 1, AssetID: nil, AccountID: nil}

	if err := worker.Work(context.Background(), &river.Job[queue_jobs.RecalculateAssetPnLArgs]{Args: args}); err == nil {
		t.Error("expected error when neither AssetID nor AccountID provided")
	}
}

func TestRecalculateAssetPnLJob_RecalcError(t *testing.T) {
	svc := &mockPnLSvc{recalcErr: errors.New("db error")}
	worker := queue_jobs.NewRecalculateAssetPnLWorker(zaptest.NewLogger(t), svc, ws.NoopBroadcaster{})
	args := queue_jobs.RecalculateAssetPnLArgs{UserID: 1, AssetID: ptr(int64(99)), AccountID: nil}

	if err := worker.Work(context.Background(), &river.Job[queue_jobs.RecalculateAssetPnLArgs]{Args: args}); err == nil {
		t.Error("expected error to propagate from RecalculateAssetPnL")
	}
}

func TestRecalculateAssetPnLJob_GetAssetIDsError(t *testing.T) {
	svc := &mockPnLSvc{assetIDsErr: errors.New("lookup failed")}
	worker := queue_jobs.NewRecalculateAssetPnLWorker(zaptest.NewLogger(t), svc, ws.NoopBroadcaster{})
	args := queue_jobs.RecalculateAssetPnLArgs{UserID: 1, AssetID: nil, AccountID: ptr(int64(5))}

	if err := worker.Work(context.Background(), &river.Job[queue_jobs.RecalculateAssetPnLArgs]{Args: args}); err == nil {
		t.Error("expected error to propagate from GetAssetIDsForAccount")
	}
}
