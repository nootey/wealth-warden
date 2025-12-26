package jobscheduler

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvestmentPriceSyncJob struct {
	logger           *zap.Logger
	container        *bootstrap.Container
	priceFetchClient finance.PriceFetcher
}

func NewInvestmentPriceSyncJob(logger *zap.Logger, container *bootstrap.Container, priceFetchClient finance.PriceFetcher) *InvestmentPriceSyncJob {
	return &InvestmentPriceSyncJob{
		logger:           logger,
		container:        container,
		priceFetchClient: priceFetchClient,
	}
}

func (j *InvestmentPriceSyncJob) Run(ctx context.Context) error {

	assets, err := j.getAssetsToUpdate(ctx)
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		j.logger.Info("No assets to update")
		return nil
	}

	priceData, err := j.fetchPrices(ctx, assets)
	if err != nil {
		return err
	}

	if len(priceData) == 0 {
		j.logger.Warn("No prices fetched successfully")
		return fmt.Errorf("no prices fetched")
	}

	updatedCount, err := j.updateAssetsAndTrades(ctx, priceData)
	if err != nil {
		return err
	}

	j.logger.Info("Investment price sync completed",
		zap.Int("assets_updated", updatedCount))

	return nil
}

func (j *InvestmentPriceSyncJob) getAssetsToUpdate(ctx context.Context) ([]struct {
	Ticker         string
	InvestmentType models.InvestmentType
}, error) {
	var assets []struct {
		Ticker         string
		InvestmentType models.InvestmentType
	}

	err := j.container.DB.WithContext(ctx).
		Model(&models.InvestmentAsset{}).
		Joins("JOIN accounts ON accounts.id = investment_assets.account_id").
		Select("DISTINCT investment_assets.ticker, investment_assets.investment_type").
		Where("investment_assets.quantity > 0").
		Where("accounts.is_active = ?", true).
		Where("accounts.closed_at IS NULL").
		Find(&assets).Error

	if err != nil {
		j.logger.Error("Failed to fetch assets", zap.Error(err))
		return nil, err
	}

	j.logger.Info("Found assets to update", zap.Int("count", len(assets)))
	return assets, nil
}

func (j *InvestmentPriceSyncJob) fetchPrices(ctx context.Context, assets []struct {
	Ticker         string
	InvestmentType models.InvestmentType
}) (map[string]*finance.PriceData, error) {

	priceData := make(map[string]*finance.PriceData)

	for i, asset := range assets {
		// Add delay between requests to avoid rate limiting
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}

		price, err := j.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err != nil {
			j.logger.Warn("Failed to fetch price",
				zap.String("ticker", asset.Ticker),
				zap.Error(err))
			continue
		}

		priceData[asset.Ticker] = price
	}

	j.logger.Info("Prices fetched", zap.Int("successful", len(priceData)))
	return priceData, nil
}

func (j *InvestmentPriceSyncJob) updateAssetsAndTrades(ctx context.Context, priceData map[string]*finance.PriceData) (int, error) {
	tx := j.container.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()
	today := now.UTC().Truncate(24 * time.Hour)
	updatedCount := 0

	// Track which accounts need balance updates
	affectedAccounts := make(map[int64]string)

	for ticker, price := range priceData {
		count, accountUpdates, err := j.updateAssetsByTicker(ctx, tx, ticker, price, now)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		updatedCount += count

		for accountID, currency := range accountUpdates {
			affectedAccounts[accountID] = currency
		}
	}

	// Update balances for all affected accounts
	for accountID, currency := range affectedAccounts {
		if err := j.updateAccountBalance(ctx, tx, accountID, currency, today); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		j.logger.Error("Failed to commit transaction", zap.Error(err))
		return 0, err
	}

	j.logger.Info("Updated balances for accounts", zap.Int("account_count", len(affectedAccounts)))

	return updatedCount, nil
}

func (j *InvestmentPriceSyncJob) updateAssetsByTicker(ctx context.Context, tx *gorm.DB, ticker string, price *finance.PriceData, now time.Time) (int, map[int64]string, error) {
	var assets []models.InvestmentAsset
	err := tx.WithContext(ctx).
		Preload("Account").
		Joins("JOIN accounts ON accounts.id = investment_assets.account_id").
		Where("investment_assets.ticker = ? AND investment_assets.quantity > 0", ticker).
		Where("accounts.is_active = ?", true).
		Where("accounts.closed_at IS NULL").
		Find(&assets).Error

	if err != nil {
		j.logger.Error("Failed to find assets",
			zap.String("ticker", ticker),
			zap.Error(err))
		return 0, nil, err
	}

	priceDecimal := decimal.NewFromFloat(price.Price)
	accountUpdates := make(map[int64]string)

	for _, asset := range assets {
		if err := j.updateAsset(tx, asset, priceDecimal, now); err != nil {
			return 0, nil, err
		}

		if err := j.updateTrades(tx, asset.ID, priceDecimal, now); err != nil {
			return 0, nil, err
		}

		// Track account that needs balance update
		accountUpdates[asset.AccountID] = asset.Account.Currency
	}

	return len(assets), accountUpdates, nil
}

func (j *InvestmentPriceSyncJob) updateAccountBalance(ctx context.Context, tx *gorm.DB, accountID int64, currency string, asOf time.Time) error {

	invService := j.container.InvestmentService

	var userID int64
	err := tx.WithContext(ctx).
		Model(&models.InvestmentAsset{}).
		Select("user_id").
		Where("account_id = ?", accountID).
		Limit(1).
		Pluck("user_id", &userID).Error

	if err != nil {
		j.logger.Error("Failed to get user ID for account",
			zap.Int64("account_id", accountID),
			zap.Error(err))
		return err
	}

	// Update balance for today only
	err = invService.UpdateInvestmentAccountBalanceRange(ctx, tx, accountID, userID, asOf, asOf, currency)
	if err != nil {
		j.logger.Error("Failed to update account balance",
			zap.Int64("account_id", accountID),
			zap.Error(err))
		return err
	}

	return nil
}

func (j *InvestmentPriceSyncJob) updateAsset(tx *gorm.DB, asset models.InvestmentAsset, price decimal.Decimal, now time.Time) error {
	newCurrentValue := asset.Quantity.Mul(price)
	newProfitLoss := newCurrentValue.Sub(asset.ValueAtBuy)
	var newProfitLossPercent decimal.Decimal
	if !asset.ValueAtBuy.IsZero() {
		newProfitLossPercent = newProfitLoss.Div(asset.ValueAtBuy)
	}

	err := tx.Model(&models.InvestmentAsset{}).
		Where("id = ?", asset.ID).
		Updates(map[string]interface{}{
			"current_price":       price,
			"current_value":       newCurrentValue,
			"profit_loss":         newProfitLoss,
			"profit_loss_percent": newProfitLossPercent,
			"last_price_update":   now,
			"updated_at":          now,
		}).Error

	if err != nil {
		j.logger.Error("Failed to update asset",
			zap.Int64("asset_id", asset.ID),
			zap.Error(err))
		return err
	}

	return nil
}

func (j *InvestmentPriceSyncJob) updateTrades(tx *gorm.DB, assetID int64, price decimal.Decimal, now time.Time) error {
	err := tx.Exec(`
        UPDATE investment_trades
        SET 
            current_value = quantity * ?,
            profit_loss = (quantity * ?) - value_at_buy,
            profit_loss_percent = CASE 
                WHEN value_at_buy > 0 THEN ((quantity * ?) - value_at_buy) / value_at_buy
                ELSE 0 
            END,
            updated_at = ?
        WHERE asset_id = ? AND trade_type = 'buy'
    `, price, price, price, now, assetID).Error

	if err != nil {
		j.logger.Error("Failed to update trades",
			zap.Int64("asset_id", assetID),
			zap.Error(err))
		return err
	}

	return nil
}
