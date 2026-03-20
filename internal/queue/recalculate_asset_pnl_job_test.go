package queue_test

import (
	"context"
	"errors"
	"testing"
	"wealth-warden/internal/queue"
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

func ptr[T any](v T) *T { return &v }

func TestRecalculateAssetPnLJob_SingleAsset(t *testing.T) {
	svc := &mockPnLSvc{}
	job := &queue.RecalculateAssetPnLJob{
		InvestmentService: svc,
		UserID:            1,
		AssetID:           ptr(int64(42)),
	}

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.recalculated) != 1 || svc.recalculated[0] != 42 {
		t.Errorf("expected asset 42 to be recalculated, got %v", svc.recalculated)
	}
}

func TestRecalculateAssetPnLJob_AccountScope(t *testing.T) {
	svc := &mockPnLSvc{assetIDs: []int64{10, 20, 30}}
	job := &queue.RecalculateAssetPnLJob{
		InvestmentService: svc,
		UserID:            1,
		AccountID:         ptr(int64(5)),
	}

	if err := job.Process(context.Background()); err != nil {
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
}

func TestRecalculateAssetPnLJob_NeitherAssetNorAccount(t *testing.T) {
	svc := &mockPnLSvc{}
	job := &queue.RecalculateAssetPnLJob{
		InvestmentService: svc,
		UserID:            1,
	}

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error when neither AssetID nor AccountID provided")
	}
}

func TestRecalculateAssetPnLJob_RecalcError(t *testing.T) {
	svc := &mockPnLSvc{recalcErr: errors.New("db error")}
	job := &queue.RecalculateAssetPnLJob{
		InvestmentService: svc,
		UserID:            1,
		AssetID:           ptr(int64(99)),
	}

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error to propagate from RecalculateAssetPnL")
	}
}

func TestRecalculateAssetPnLJob_GetAssetIDsError(t *testing.T) {
	svc := &mockPnLSvc{assetIDsErr: errors.New("lookup failed")}
	job := &queue.RecalculateAssetPnLJob{
		InvestmentService: svc,
		UserID:            1,
		AccountID:         ptr(int64(5)),
	}

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error to propagate from GetAssetIDsForAccount")
	}
}
