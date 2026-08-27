package jobs

import (
	"context"
	"fmt"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type pnlSvc interface {
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
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

	if args.AssetID != nil {
		w.logger.Info("Recalculating asset PnL", zap.Int64("assetID", *args.AssetID))
		if err := w.investmentService.RecalculateAssetPnL(ctx, args.UserID, *args.AssetID); err != nil {
			w.logger.Error("Failed to recalculate asset PnL", zap.Int64("assetID", *args.AssetID), zap.Error(err))
			return fmt.Errorf("failed to recalculate PnL for asset %d: %w", *args.AssetID, err)
		}
		w.logger.Info("Asset PnL recalculated", zap.Int64("assetID", *args.AssetID))
		w.refreshSnapshots(ctx, args.UserID)
		w.broadcaster.Send(args.UserID, ws.Event{Type: ws.TypeAssetPnLSynced, Payload: ws.AssetPnLPayload{AssetID: args.AssetID}})
		return nil
	}

	if args.AccountID != nil {
		w.logger.Info("Recalculating PnL for all assets in account", zap.Int64("accountID", *args.AccountID))
		assetIDs, err := w.investmentService.GetAssetIDsForAccount(ctx, args.UserID, *args.AccountID)
		if err != nil {
			w.logger.Error("Failed to fetch assets for account", zap.Int64("accountID", *args.AccountID), zap.Error(err))
			return fmt.Errorf("failed to get assets for account %d: %w", *args.AccountID, err)
		}
		for _, id := range assetIDs {
			if err := w.investmentService.RecalculateAssetPnL(ctx, args.UserID, id); err != nil {
				w.logger.Error("Failed to recalculate asset PnL", zap.Int64("assetID", id), zap.Error(err))
				return fmt.Errorf("failed to recalculate PnL for asset %d: %w", id, err)
			}
		}
		w.logger.Info("Account PnL recalculated", zap.Int64("accountID", *args.AccountID), zap.Int("assets", len(assetIDs)))
		w.refreshSnapshots(ctx, args.UserID)
		w.broadcaster.Send(args.UserID, ws.Event{Type: ws.TypeAssetPnLSynced, Payload: ws.AssetPnLPayload{AccountID: args.AccountID}})
		return nil
	}

	return fmt.Errorf("recalculate asset PnL: neither AssetID nor AccountID provided")
}

func (w *RecalculateAssetPnLWorker) refreshSnapshots(ctx context.Context, userID int64) {
	if err := w.investmentService.UpdateSnapshotMarketValues(ctx, userID); err != nil {
		w.logger.Warn("Failed to refresh snapshot market values", zap.Int64("userID", userID), zap.Error(err))
	}
}
