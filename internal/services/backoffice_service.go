package services

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

type BackofficeServiceInterface interface {
	BackfillAssetCashFlows(ctx context.Context) error
	CorrectFeeAccounting(ctx context.Context) error
	PreviewZeroCostTradeMigration(ctx context.Context) (*models.ZeroCostTradeMigrationPreview, error)
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
	return s.jobDispatcher.Dispatch(queue.NewBackfillAssetCashFlowsJob(
		s.logger.Named("backfill_asset_cash_flows"),
		s.investmentService,
		s.accountService,
		s.userService,
	))
}

func (s *BackofficeService) CorrectFeeAccounting(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(queue.NewCorrectFeeAccountingJob(
		s.logger.Named("correct_fee_accounting"),
		s.investmentService,
		s.accountService,
		s.userService,
	))
}

func (s *BackofficeService) PreviewZeroCostTradeMigration(ctx context.Context) (*models.ZeroCostTradeMigrationPreview, error) {
	trades, err := s.repo.GetZeroCostBuyTrades(ctx)
	if err != nil {
		return nil, err
	}

	assetGroups := make(map[int64]*models.ZeroCostTradeAssetGroup)
	assetOrder := []int64{}

	for _, trade := range trades {
		if _, ok := assetGroups[trade.AssetID]; !ok {
			incomeType := models.IncomeTypeStaking
			if trade.Asset.InvestmentType != models.InvestmentCrypto {
				incomeType = models.IncomeTypeDividend
			}
			assetGroups[trade.AssetID] = &models.ZeroCostTradeAssetGroup{
				AssetID:        trade.AssetID,
				Ticker:         trade.Asset.Ticker,
				AssetName:      trade.Asset.Name,
				InvestmentType: trade.Asset.InvestmentType,
				IncomeType:     incomeType,
			}
			assetOrder = append(assetOrder, trade.AssetID)
		}
		group := assetGroups[trade.AssetID]
		group.Trades = append(group.Trades, models.ZeroCostTradePreview{
			ID:       trade.ID,
			TxnDate:  trade.TxnDate,
			Quantity: trade.Quantity,
			Currency: trade.Currency,
		})
		group.TradeCount++
	}

	for _, assetID := range assetOrder {
		group := assetGroups[assetID]
		s.logger.Info("zero-cost trade migration preview",
			zap.Int64("asset_id", group.AssetID),
			zap.String("ticker", group.Ticker),
			zap.String("investment_type", string(group.InvestmentType)),
			zap.String("income_type", string(group.IncomeType)),
			zap.Int("trade_count", group.TradeCount),
		)
		for _, t := range group.Trades {
			s.logger.Info("  trade",
				zap.Int64("id", t.ID),
				zap.String("date", t.TxnDate.Format("2006-01-02")),
				zap.String("quantity", t.Quantity.String()),
				zap.String("currency", t.Currency),
			)
		}
	}

	result := &models.ZeroCostTradeMigrationPreview{
		TotalTrades: len(trades),
		AssetCount:  len(assetGroups),
	}
	for _, assetID := range assetOrder {
		result.Assets = append(result.Assets, *assetGroups[assetID])
	}

	return result, nil
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
