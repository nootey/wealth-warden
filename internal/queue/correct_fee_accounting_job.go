package queue

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type feeAccountingCorrectionSvc interface {
	CorrectTradeFeeAccounting(ctx context.Context, userID int64) error
	BackfillInvestmentCashFlows(ctx context.Context, userID int64) error
}

type CorrectFeeAccountingJob struct {
	logger            *zap.Logger
	InvestmentService feeAccountingCorrectionSvc `json:"-"`
	AccountService    accountBackfillSvc         `json:"-"`
	UserService       userBackfillSvc            `json:"-"`
}

func (j *CorrectFeeAccountingJob) Type() string { return TypeCorrectFeeAccounting }

func NewCorrectFeeAccountingJob(
	logger *zap.Logger,
	investmentService feeAccountingCorrectionSvc,
	accountService accountBackfillSvc,
	userService userBackfillSvc,
) *CorrectFeeAccountingJob {
	return &CorrectFeeAccountingJob{
		logger:            logger,
		InvestmentService: investmentService,
		AccountService:    accountService,
		UserService:       userService,
	}
}

func (j *CorrectFeeAccountingJob) Process(ctx context.Context) error {
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
		// Step 1: fix trade value_at_buy and recalculate asset aggregates
		if err := j.InvestmentService.CorrectTradeFeeAccounting(ctx, userID); err != nil {
			j.logger.Error("Failed to correct trade fee accounting", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		// Step 2: clear and rebuild cash flows with corrected values
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

		if err := j.InvestmentService.BackfillInvestmentCashFlows(ctx, userID); err != nil {
			j.logger.Error("Failed to backfill cash flows", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		if err := j.AccountService.RebuildSnapshotsForUser(ctx, userID); err != nil {
			j.logger.Error("Failed to rebuild snapshots", zap.Int64("userID", userID), zap.Error(err))
			failCount++
			continue
		}

		j.logger.Info("User correction complete", zap.Int64("userID", userID))
	}

	j.logger.Info("Completed", zap.Int("success", len(userIDs)-failCount), zap.Int("failed", failCount))
	return nil
}
