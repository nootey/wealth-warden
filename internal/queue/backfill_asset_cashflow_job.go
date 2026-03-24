package queue

import (
	"context"
	"fmt"

	"go.uber.org/zap"
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
	logger            *zap.Logger
	InvestmentService investmentBackfillSvc
	AccountService    accountBackfillSvc
	UserService       userBackfillSvc
}

func NewBackfillAssetCashFlowsJob(
	logger *zap.Logger,
	investmentService investmentBackfillSvc,
	accountService accountBackfillSvc,
	userService userBackfillSvc,
) *BackfillAssetCashFlowsJob {
	return &BackfillAssetCashFlowsJob{
		logger:            logger,
		InvestmentService: investmentService,
		AccountService:    accountService,
		UserService:       userService,
	}
}

func (j *BackfillAssetCashFlowsJob) Process(ctx context.Context) error {
	userIDs, err := j.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		j.logger.Info("No users to process")
		return nil
	}

	j.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	failCount := 0

	for _, userID := range userIDs {

		// Clear derived data, keep raw ledger and trades
		if err := j.AccountService.ClearInvestmentCashFlows(ctx, userID); err != nil {
			j.logger.Error("Failed to clear investment cash flows", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		if err := j.AccountService.ClearInvestmentSnapshots(ctx, userID); err != nil {
			j.logger.Error("Failed to clear investment snapshots", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		// Add investment trade cash flows on top of the reset balance rows
		if err := j.InvestmentService.BackfillInvestmentCashFlows(ctx, userID); err != nil {
			j.logger.Error("Failed to backfill cash flows", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		// Frontfill start_balance chains then rebuild all snapshots from scratch
		if err := j.AccountService.RebuildSnapshotsForUser(ctx, userID); err != nil {
			j.logger.Error("Failed to rebuild snapshots", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		j.logger.Info("User backfill complete", zap.Int64("userID", userID))
	}

	j.logger.Info("Completed", zap.Int("success", len(userIDs)-failCount), zap.Int("failed", failCount))
	return nil
}
