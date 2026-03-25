package queue

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type pnlSvc interface {
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
}

type RecalculateAssetPnLJob struct {
	logger            *zap.Logger
	InvestmentService pnlSvc
	UserID            int64
	AssetID           *int64 // nil = all assets for the account
	AccountID         *int64 // nil = single asset mode
}

func NewRecalculateAssetPnLJob(
	logger *zap.Logger,
	investmentService pnlSvc,
	userID int64,
	assetID *int64,
	accountID *int64,
) *RecalculateAssetPnLJob {
	return &RecalculateAssetPnLJob{
		logger:            logger,
		InvestmentService: investmentService,
		UserID:            userID,
		AssetID:           assetID,
		AccountID:         accountID,
	}
}

func (j *RecalculateAssetPnLJob) Process(ctx context.Context) error {
	if j.AssetID != nil {
		j.logger.Info("Recalculating asset PnL", zap.Int64("assetID", *j.AssetID))
		if err := j.InvestmentService.RecalculateAssetPnL(ctx, j.UserID, *j.AssetID); err != nil {
			j.logger.Error("Failed to recalculate asset PnL", zap.Int64("assetID", *j.AssetID), zap.Error(err))
			return fmt.Errorf("failed to recalculate PnL for asset %d: %w", *j.AssetID, err)
		}
		j.logger.Info("Asset PnL recalculated", zap.Int64("assetID", *j.AssetID))
		j.refreshSnapshots(ctx)
		return nil
	}

	if j.AccountID != nil {
		j.logger.Info("Recalculating PnL for all assets in account", zap.Int64("accountID", *j.AccountID))
		assetIDs, err := j.InvestmentService.GetAssetIDsForAccount(ctx, j.UserID, *j.AccountID)
		if err != nil {
			j.logger.Error("Failed to fetch assets for account", zap.Int64("accountID", *j.AccountID), zap.Error(err))
			return fmt.Errorf("failed to get assets for account %d: %w", *j.AccountID, err)
		}
		for _, id := range assetIDs {
			if err := j.InvestmentService.RecalculateAssetPnL(ctx, j.UserID, id); err != nil {
				j.logger.Error("Failed to recalculate asset PnL", zap.Int64("assetID", id), zap.Error(err))
				return fmt.Errorf("failed to recalculate PnL for asset %d: %w", id, err)
			}
		}
		j.logger.Info("Account PnL recalculated", zap.Int64("accountID", *j.AccountID), zap.Int("assets", len(assetIDs)))
		j.refreshSnapshots(ctx)
		return nil
	}

	return fmt.Errorf("RecalculateAssetPnLJob: neither AssetID nor AccountID provided")
}

func (j *RecalculateAssetPnLJob) refreshSnapshots(ctx context.Context) {
	if err := j.InvestmentService.UpdateSnapshotMarketValues(ctx, j.UserID); err != nil {
		j.logger.Warn("Failed to refresh snapshot market values", zap.Int64("userID", j.UserID), zap.Error(err))
	}
}
