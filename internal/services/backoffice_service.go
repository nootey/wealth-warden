package services

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

type BackofficeServiceInterface interface {
	BackfillAssetCashFlows(ctx context.Context) error
	CorrectFeeAccounting(ctx context.Context) error
	MigrateZeroCostTrades(ctx context.Context) (*models.ZeroCostMigrationResult, error)
}

type BackofficeService struct {
	logger            *zap.Logger
	jobDispatcher     queue.JobDispatcher
	repo              repositories.BackofficeRepositoryInterface
	investmentService InvestmentServiceInterface
	accountService    AccountServiceInterface
	userService       UserServiceInterface
}

func NewBackofficeService(
	logger *zap.Logger,
	jobDispatcher queue.JobDispatcher,
	repo *repositories.BackofficeRepository,
	investmentService InvestmentServiceInterface,
	accountService AccountServiceInterface,
	userService UserServiceInterface,
) *BackofficeService {
	return &BackofficeService{
		logger:            logger,
		jobDispatcher:     jobDispatcher,
		repo:              repo,
		investmentService: investmentService,
		accountService:    accountService,
		userService:       userService,
	}
}

var _ BackofficeServiceInterface = (*BackofficeService)(nil)

func (s *BackofficeService) BackfillAssetCashFlows(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, queue_jobs.NewBackfillAssetCashFlowsJob(
		s.logger.Named("backfill_asset_cash_flows"),
		s.investmentService,
		s.accountService,
		s.userService,
	))
}

func (s *BackofficeService) CorrectFeeAccounting(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, queue_jobs.NewCorrectFeeAccountingJob(
		s.logger.Named("correct_fee_accounting"),
		s.investmentService,
		s.accountService,
		s.userService,
	))
}

func (s *BackofficeService) MigrateZeroCostTrades(ctx context.Context) (*models.ZeroCostMigrationResult, error) {
	trades, err := s.repo.GetZeroCostBuyTrades(ctx)
	if err != nil {
		return nil, err
	}

	assetGroups := make(map[int64][]models.InvestmentTrade)
	assetOrder := []int64{}
	assetTicker := make(map[int64]string)

	for _, trade := range trades {
		if _, ok := assetGroups[trade.AssetID]; !ok {
			assetOrder = append(assetOrder, trade.AssetID)
			assetTicker[trade.AssetID] = trade.Asset.Ticker
		}
		assetGroups[trade.AssetID] = append(assetGroups[trade.AssetID], trade)
	}

	result := &models.ZeroCostMigrationResult{}

	for _, assetID := range assetOrder {
		group := assetGroups[assetID]
		userID := group[0].UserID

		s.logger.Info("migrating zero-cost trades for asset",
			zap.Int64("asset_id", assetID),
			zap.String("ticker", assetTicker[assetID]),
			zap.Int("trade_count", len(group)),
		)

		if err := s.investmentService.MigrateZeroCostTradesForAsset(ctx, userID, assetID, group); err != nil {
			s.logger.Error("failed to migrate trades for asset",
				zap.Int64("asset_id", assetID),
				zap.String("ticker", assetTicker[assetID]),
				zap.Error(err),
			)
			result.AssetsFailed++
			result.Errors = append(result.Errors, models.ZeroCostMigrationError{
				AssetID: assetID,
				Ticker:  assetTicker[assetID],
				Error:   err.Error(),
			})
			continue
		}

		result.TotalProcessed += len(group)
		result.AssetsProcessed++
	}

	return result, nil
}
