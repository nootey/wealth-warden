package queue

import (
	"context"
	"fmt"
)

type pnlInvestmentSvc interface {
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
}

type RecalculateAssetPnLJob struct {
	InvestmentService pnlInvestmentSvc
	UserID            int64
	AssetID           *int64 // nil = all assets for the account
	AccountID         *int64 // nil = single asset mode
}

func (j *RecalculateAssetPnLJob) Process(ctx context.Context) error {
	if j.AssetID != nil {
		fmt.Printf("pnl sync: recalculating asset %d\n", *j.AssetID)
		if err := j.InvestmentService.RecalculateAssetPnL(ctx, j.UserID, *j.AssetID); err != nil {
			fmt.Printf("pnl sync: error recalculating asset %d: %v\n", *j.AssetID, err)
			return fmt.Errorf("failed to recalculate PnL for asset %d: %w", *j.AssetID, err)
		}
		fmt.Printf("pnl sync: asset %d done\n", *j.AssetID)
		return nil
	}

	if j.AccountID != nil {
		fmt.Printf("pnl sync: recalculating all assets for account %d\n", *j.AccountID)
		assetIDs, err := j.InvestmentService.GetAssetIDsForAccount(ctx, j.UserID, *j.AccountID)
		if err != nil {
			fmt.Printf("pnl sync: error fetching assets for account %d: %v\n", *j.AccountID, err)
			return fmt.Errorf("failed to get assets for account %d: %w", *j.AccountID, err)
		}
		for _, id := range assetIDs {
			if err := j.InvestmentService.RecalculateAssetPnL(ctx, j.UserID, id); err != nil {
				fmt.Printf("pnl sync: error recalculating asset %d: %v\n", id, err)
				return fmt.Errorf("failed to recalculate PnL for asset %d: %w", id, err)
			}
		}
		fmt.Printf("pnl sync: account %d done (%d assets)\n", *j.AccountID, len(assetIDs))
		return nil
	}

	return fmt.Errorf("RecalculateAssetPnLJob: neither AssetID nor AccountID provided")
}
