package queue

import (
	"context"
	"fmt"
)

type investmentBackfillSvc interface {
	BackfillInvestmentCashFlows(ctx context.Context, userID int64) error
}

type accountBackfillSvc interface {
	ClearInvestmentCashFlows(ctx context.Context, userID int64) error
	ClearInvestmentSnapshots(ctx context.Context, userID int64) error
	RebuildSnapshotsForUser(ctx context.Context, userID int64) error
}

type userBackfillSvc interface {
	GetAllActiveUserIDs(ctx context.Context) ([]int64, error)
}

type BackfillAssetCashFlowsJob struct {
	InvestmentService investmentBackfillSvc
	AccountService    accountBackfillSvc
	UserService       userBackfillSvc
}

func (j *BackfillAssetCashFlowsJob) Process(ctx context.Context) error {
	userIDs, err := j.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		fmt.Println("asset cash backfill: no users to process")
		return nil
	}

	fmt.Printf("asset cash backfill: processing %d users\n", len(userIDs))

	for _, userID := range userIDs {

		// Clear derived data, keep raw ledger and trades
		if err := j.AccountService.ClearInvestmentCashFlows(ctx, userID); err != nil {
			return fmt.Errorf("failed to clear investment cash flows for user %d: %w", userID, err)
		}

		if err := j.AccountService.ClearInvestmentSnapshots(ctx, userID); err != nil {
			return fmt.Errorf("failed to clear investment snapshots for user %d: %w", userID, err)
		}

		// Add investment trade cash flows on top of the reset balance rows
		if err := j.InvestmentService.BackfillInvestmentCashFlows(ctx, userID); err != nil {
			return fmt.Errorf("failed to backfill cash flows for user %d: %w", userID, err)
		}

		// Frontfill start_balance chains then rebuild all snapshots from scratch
		if err := j.AccountService.RebuildSnapshotsForUser(ctx, userID); err != nil {
			return fmt.Errorf("failed to rebuild snapshots for user %d: %w", userID, err)
		}
	}

	fmt.Println("asset cash backfill: completed successfully")
	return nil
}
