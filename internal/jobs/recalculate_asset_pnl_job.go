package jobs

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type pnlSvc interface {
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	UpdateSnapshotMarketValues(ctx context.Context, userID int64, from time.Time) error
}

type RecalculateAssetPnLWorker struct {
	river.WorkerDefaults[jobqueue.RecalculateAssetPnLArgs]
	logger            *zap.Logger
	investmentService pnlSvc
	broadcaster       ws.Broadcaster
}

func NewRecalculateAssetPnLWorker(logger *zap.Logger, investmentService pnlSvc, broadcaster ws.Broadcaster) *RecalculateAssetPnLWorker {
	return &RecalculateAssetPnLWorker{logger: logger, investmentService: investmentService, broadcaster: broadcaster}
}

func (w *RecalculateAssetPnLWorker) Work(ctx context.Context, job *river.Job[jobqueue.RecalculateAssetPnLArgs]) error {
	args := job.Args

	assetIDs, payload, err := w.resolveScope(ctx, args)
	if err != nil {
		return err
	}

	var failed int
	for _, id := range assetIDs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := w.investmentService.RecalculateAssetPnL(ctx, args.UserID, id); err != nil {
			if failed == 0 {
				w.logger.Error("Failed to recalculate asset PnL", zap.Int64("assetID", id), zap.Error(err))
			}
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("failed to recalculate PnL for %d of %d assets", failed, len(assetIDs))
	}

	w.logger.Info("Asset PnL recalculated", zap.Int("assets", len(assetIDs)))
	w.refreshSnapshots(ctx, args.UserID)
	w.broadcaster.Send(args.UserID, ws.Event{Type: ws.TypeAssetPnLSynced, Payload: payload})
	return nil
}

func (w *RecalculateAssetPnLWorker) resolveScope(ctx context.Context, args jobqueue.RecalculateAssetPnLArgs) ([]int64, ws.AssetPnLPayload, error) {
	switch {
	case args.AssetID != nil:
		w.logger.Info("Recalculating asset PnL", zap.Int64("assetID", *args.AssetID))
		return []int64{*args.AssetID}, ws.AssetPnLPayload{AssetID: args.AssetID}, nil

	case args.AccountID != nil:
		w.logger.Info("Recalculating PnL for all assets in account", zap.Int64("accountID", *args.AccountID))
		assetIDs, err := w.investmentService.GetAssetIDsForAccount(ctx, args.UserID, *args.AccountID)
		if err != nil {
			w.logger.Error("Failed to fetch assets for account", zap.Int64("accountID", *args.AccountID), zap.Error(err))
			return nil, ws.AssetPnLPayload{}, fmt.Errorf("failed to get assets for account %d: %w", *args.AccountID, err)
		}
		return assetIDs, ws.AssetPnLPayload{AccountID: args.AccountID}, nil

	default:
		return nil, ws.AssetPnLPayload{}, fmt.Errorf("recalculate asset PnL: neither AssetID nor AccountID provided")
	}
}

func (w *RecalculateAssetPnLWorker) refreshSnapshots(ctx context.Context, userID int64) {
	// A PnL recalc can restate held quantity on any historical day (back-dated
	// trade edits), so the whole snapshot series is recomputed here.
	if err := w.investmentService.UpdateSnapshotMarketValues(ctx, userID, time.Time{}); err != nil {
		w.logger.Warn("Failed to refresh snapshot market values", zap.Int64("userID", userID), zap.Error(err))
	}
}
