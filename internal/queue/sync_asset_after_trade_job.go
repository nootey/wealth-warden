package queue

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"go.uber.org/zap"
)

type postTradeSyncSvc interface {
	BackfillAssetPriceHistory(ctx context.Context, assetID int64, ticker string, investmentType models.InvestmentType, from, to time.Time) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
}

type SyncAssetAfterTradeJob struct {
	logger            *zap.Logger
	InvestmentService postTradeSyncSvc
	UserID            int64
	AssetID           int64
	Ticker            string
	InvestmentType    models.InvestmentType
	TradeDate         time.Time
}

func NewSyncAssetAfterTradeJob(
	logger *zap.Logger,
	investmentService postTradeSyncSvc,
	userID, assetID int64,
	ticker string,
	investmentType models.InvestmentType,
	tradeDate time.Time,
) *SyncAssetAfterTradeJob {
	return &SyncAssetAfterTradeJob{
		logger:            logger,
		InvestmentService: investmentService,
		UserID:            userID,
		AssetID:           assetID,
		Ticker:            ticker,
		InvestmentType:    investmentType,
		TradeDate:         tradeDate,
	}
}

func (j *SyncAssetAfterTradeJob) Process(ctx context.Context) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := j.InvestmentService.BackfillAssetPriceHistory(ctx, j.AssetID, j.Ticker, j.InvestmentType, j.TradeDate, today); err != nil {
		j.logger.Warn("Failed to backfill asset price history",
			zap.Int64("assetID", j.AssetID),
			zap.String("ticker", j.Ticker),
			zap.Error(err),
		)
	}

	if err := j.InvestmentService.UpdateSnapshotMarketValues(ctx, j.UserID); err != nil {
		j.logger.Warn("Failed to update snapshot market values",
			zap.Int64("userID", j.UserID),
			zap.Error(err),
		)
	}

	return nil
}
