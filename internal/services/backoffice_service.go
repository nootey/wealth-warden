package services

import (
	"context"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

type BackofficeServiceInterface interface {
	BackfillAssetCashFlows(ctx context.Context) error
	CorrectFeeAccounting(ctx context.Context) error
	BackfillIncomeExchangeRates(ctx context.Context) error
	MigrateZeroCostTrades(ctx context.Context) error
	RunZeroCostTradeMigration(ctx context.Context) (*models.ZeroCostMigrationResult, error)
}

type BackofficeService struct {
	logger            *zap.Logger
	jobDispatcher     jobqueue.Dispatcher
	repo              repositories.BackofficeRepositoryInterface
	investmentService InvestmentServiceInterface
}

func NewBackofficeService(
	logger *zap.Logger,
	jobDispatcher jobqueue.Dispatcher,
	repo *repositories.BackofficeRepository,
	investmentService InvestmentServiceInterface,
) *BackofficeService {
	return &BackofficeService{
		logger:            logger,
		jobDispatcher:     jobDispatcher,
		repo:              repo,
		investmentService: investmentService,
	}
}

var _ BackofficeServiceInterface = (*BackofficeService)(nil)

func (s *BackofficeService) BackfillAssetCashFlows(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.BackfillAssetCashFlowsArgs{})
}

func (s *BackofficeService) CorrectFeeAccounting(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.CorrectFeeAccountingArgs{})
}

func (s *BackofficeService) BackfillIncomeExchangeRates(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.BackfillIncomeFXRatesArgs{})
}

func (s *BackofficeService) MigrateZeroCostTrades(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.MigrateZeroCostTradesArgs{})
}

func (s *BackofficeService) RunZeroCostTradeMigration(ctx context.Context) (*models.ZeroCostMigrationResult, error) {
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
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

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
			continue
		}

		result.TotalProcessed += len(group)
		result.AssetsProcessed++
	}

	return result, nil
}
