package jobs

import (
	"context"
	"errors"
	"time"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type postTradeSyncSvc interface {
	BackfillAssetPriceHistory(ctx context.Context, assetID int64, ticker string, from, to time.Time) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
}

type SyncAssetAfterTradeWorker struct {
	river.WorkerDefaults[jobqueue.SyncAssetAfterTradeArgs]
	logger            *zap.Logger
	investmentService postTradeSyncSvc
}

func NewSyncAssetAfterTradeWorker(logger *zap.Logger, investmentService postTradeSyncSvc) *SyncAssetAfterTradeWorker {
	return &SyncAssetAfterTradeWorker{logger: logger, investmentService: investmentService}
}

func (w *SyncAssetAfterTradeWorker) Work(ctx context.Context, job *river.Job[jobqueue.SyncAssetAfterTradeArgs]) error {
	args := job.Args
	today := time.Now().UTC().Truncate(24 * time.Hour)

	var errs []error

	if err := w.investmentService.BackfillAssetPriceHistory(ctx, args.AssetID, args.Ticker, args.TradeDate, today); err != nil {
		w.logger.Warn("Failed to backfill asset price history",
			zap.Int64("assetID", args.AssetID),
			zap.String("ticker", args.Ticker),
			zap.Error(err),
		)
		errs = append(errs, err)
	}

	if err := w.investmentService.UpdateSnapshotMarketValues(ctx, args.UserID); err != nil {
		w.logger.Warn("Failed to update snapshot market values",
			zap.Int64("userID", args.UserID),
			zap.Error(err),
		)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
