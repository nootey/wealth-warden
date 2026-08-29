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
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubPostTradeSync struct {
	backfillErr error
	snapshotErr error
	backfilled  bool
	snapshotted bool
}

func (s *stubPostTradeSync) BackfillTickerPriceHistory(context.Context, string, time.Time, time.Time) error {
	s.backfilled = true
	return s.backfillErr
}

func (s *stubPostTradeSync) UpdateSnapshotMarketValues(context.Context, int64, time.Time) error {
	s.snapshotted = true
	return s.snapshotErr
}

func runSyncAssetJob(t *testing.T, svc *stubPostTradeSync) error {
	t.Helper()
	worker := jobs.NewSyncAssetAfterTradeWorker(zap.NewNop(), svc)
	return worker.Work(context.Background(), &river.Job[jobqueue.SyncAssetAfterTradeArgs]{
		Args: jobqueue.SyncAssetAfterTradeArgs{
			UserID: 1, AssetID: 2, Ticker: "IWDA.AS",
			InvestmentType: models.InvestmentStock,
			TradeDate:      time.Now().UTC().AddDate(0, 0, -3),
		},
	})
}

func TestSyncAssetAfterTrade_BackfillFailureStillUpdatesSnapshot(t *testing.T) {
	backfillErr := errors.New("price api down")
	svc := &stubPostTradeSync{backfillErr: backfillErr}

	err := runSyncAssetJob(t, svc)

	require.True(t, svc.snapshotted)
	require.ErrorIs(t, err, backfillErr)
}

func TestSyncAssetAfterTrade_ReturnsBothErrors(t *testing.T) {
	backfillErr := errors.New("price api down")
	snapshotErr := errors.New("db down")
	svc := &stubPostTradeSync{backfillErr: backfillErr, snapshotErr: snapshotErr}

	err := runSyncAssetJob(t, svc)

	require.ErrorIs(t, err, backfillErr)
	require.ErrorIs(t, err, snapshotErr)
}

func TestSyncAssetAfterTrade_SuccessReturnsNil(t *testing.T) {
	svc := &stubPostTradeSync{}

	require.NoError(t, runSyncAssetJob(t, svc))
	require.True(t, svc.backfilled)
	require.True(t, svc.snapshotted)
}
