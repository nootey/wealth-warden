package jobscheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type accService interface {
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
}

const priceSurgeThreshold = 0.10

type AssetPriceSyncJob struct {
	logger            *zap.Logger
	investmentSvc     services.InvestmentServiceInterface
	accService        accService
	db                *gorm.DB
	priceFetchClient  finance.PriceFetcher
	notifDispatcher   queue.NotificationDispatcher
	concurrentWorkers int
}

func NewAssetPriceSyncJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	accService accService,
	db *gorm.DB,
	priceFetchClient finance.PriceFetcher,
	notifDispatcher queue.NotificationDispatcher,
	concurrentWorkers int,
) *AssetPriceSyncJob {
	return &AssetPriceSyncJob{
		logger:            logger,
		investmentSvc:     investmentSvc,
		accService:        accService,
		db:                db,
		priceFetchClient:  priceFetchClient,
		notifDispatcher:   notifDispatcher,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AssetPriceSyncJob) Run(ctx context.Context) error {

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

	j.logger.Info("Asset price sync completed",
		zap.Int("assets_updated", updatedCount))

	if err := j.refreshSnapshotMarketValues(ctx); err != nil {
		j.logger.Warn("Failed to refresh snapshot market values after price sync", zap.Error(err))
	}

	return nil
}

func (j *AssetPriceSyncJob) refreshSnapshotMarketValues(ctx context.Context) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Warm today's exchange rates for all active price→account currency pairs
	// so the SQL in UpdateSnapshotMarketValues has fresh rates to work with.
	type currencyPair struct {
		FromCurrency string
		ToCurrency   string
	}
	var pairs []currencyPair
	if err := j.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ph.currency AS from_currency, a.currency AS to_currency
		FROM investment_assets ia
		JOIN accounts a ON a.id = ia.account_id
		JOIN asset_price_history ph ON ph.asset_id = ia.id
		WHERE a.is_active = TRUE AND a.closed_at IS NULL
		  AND ph.currency != a.currency
	`).Scan(&pairs).Error; err != nil {
		j.logger.Warn("Failed to query currency pairs for rate refresh", zap.Error(err))
	}
	for _, p := range pairs {
		if _, err := j.investmentSvc.GetExchangeRate(ctx, p.FromCurrency, p.ToCurrency, &today); err != nil {
			j.logger.Warn("Failed to refresh exchange rate",
				zap.String("from", p.FromCurrency),
				zap.String("to", p.ToCurrency),
				zap.Error(err))
		}
	}

	var userIDs []int64
	err := j.db.WithContext(ctx).Raw(`
		SELECT DISTINCT a.user_id
		FROM investment_assets ia
		JOIN accounts a ON a.id = ia.account_id
		WHERE a.is_active = TRUE AND a.closed_at IS NULL
	`).Scan(&userIDs).Error
	if err != nil {
		return err
	}

	jobs := make(chan int64, len(userIDs))
	var wg sync.WaitGroup
	for i := 0; i < j.concurrentWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uid := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err := j.accService.UpdateSnapshotMarketValues(ctx, uid); err != nil {
					j.logger.Warn("Failed to update snapshot market values",
						zap.Int64("userID", uid),
						zap.Error(err))
				}
			}
		}()
	}

	for _, uid := range userIDs {
		jobs <- uid
	}
	close(jobs)
	wg.Wait()

	return nil
}

func (j *AssetPriceSyncJob) getAssetsToUpdate(ctx context.Context) ([]struct {
	Ticker         string
	InvestmentType models.InvestmentType
}, error) {
	var assets []struct {
		Ticker         string
		InvestmentType models.InvestmentType
	}

	err := j.db.WithContext(ctx).
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

func (j *AssetPriceSyncJob) fetchPrices(ctx context.Context, assets []struct {
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

		if price == nil || price.Price <= 0 {
			j.logger.Error("Invalid price received",
				zap.String("ticker", asset.Ticker),
				zap.Float64("price", price.Price))
			continue
		}

		priceData[asset.Ticker] = price
	}

	j.logger.Info("Prices fetched", zap.Int("successful", len(priceData)))

	failedCount := len(assets) - len(priceData)
	if failedCount > 0 {
		j.logger.Warn("Some prices failed to fetch",
			zap.Int("failed_count", failedCount),
			zap.Int("total_assets", len(assets)))
	}

	return priceData, nil
}

func (j *AssetPriceSyncJob) updateAssetsAndTrades(ctx context.Context, priceData map[string]*finance.PriceData) (int, error) {
	tx := j.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()
	today := now.UTC().Truncate(24 * time.Hour)
	updatedCount := 0

	for ticker, price := range priceData {
		count, err := j.updateAssetsByTicker(ctx, tx, ticker, price, now, today)
		if err != nil {
			j.logger.Error("Failed to update assets for ticker",
				zap.String("ticker", ticker),
				zap.Error(err))
			tx.Rollback()
			return 0, err
		}
		updatedCount += count
	}

	if err := tx.Commit().Error; err != nil {
		j.logger.Error("Failed to commit transaction", zap.Error(err))
		return 0, err
	}

	return updatedCount, nil
}

func (j *AssetPriceSyncJob) updateAssetsByTicker(ctx context.Context, tx *gorm.DB, ticker string, price *finance.PriceData, now time.Time, today time.Time) (int, error) {
	var assets []models.InvestmentAsset
	err := tx.WithContext(ctx).
		Preload("Account").
		Joins("JOIN accounts ON accounts.id = investment_assets.account_id").
		Where("investment_assets.ticker = ? AND investment_assets.quantity > 0", ticker).
		Where("accounts.is_active = ?", true).
		Where("accounts.closed_at IS NULL").
		Find(&assets).Error

	if err != nil {
		return 0, err
	}

	priceDecimal := decimal.NewFromFloat(price.Price)

	for _, asset := range assets {
		if err := j.updateAsset(tx, asset, priceDecimal, now); err != nil {
			return 0, err
		}

		if err := j.updateTrades(tx, asset.ID, priceDecimal, now); err != nil {
			return 0, err
		}

		// Persist to price history
		if err := j.investmentSvc.UpsertAssetPrice(ctx, tx, []models.AssetPriceHistory{{AssetID: asset.ID, AsOf: today, Price: priceDecimal, Currency: price.Currency}}); err != nil {
			j.logger.Warn("Failed to upsert asset price history",
				zap.Int64("asset_id", asset.ID),
				zap.Error(err))
			// non-fatal, continue
		}
	}

	return len(assets), nil
}

func (j *AssetPriceSyncJob) updateAsset(tx *gorm.DB, asset models.InvestmentAsset, price decimal.Decimal, now time.Time) error {

	if price.IsZero() || price.IsNegative() {
		j.logger.Error("Refusing to update asset with invalid price",
			zap.Int64("asset_id", asset.ID),
			zap.String("ticker", asset.Ticker),
			zap.String("price", price.String()))
		return fmt.Errorf("invalid price for asset %d: %s", asset.ID, price.String())
	}

	if asset.CurrentPrice != nil && !asset.CurrentPrice.IsZero() {
		changePercent := price.Sub(*asset.CurrentPrice).Div(*asset.CurrentPrice).Abs()
		if changePercent.GreaterThan(decimal.NewFromFloat(0.90)) && price.LessThan(*asset.CurrentPrice) {
			j.logger.Warn("Extreme price drop detected — skipping update to prevent data corruption",
				zap.Int64("asset_id", asset.ID),
				zap.String("ticker", asset.Ticker),
				zap.String("old_price", asset.CurrentPrice.String()),
				zap.String("new_price", price.String()),
				zap.String("change_percent", changePercent.Mul(decimal.NewFromInt(100)).StringFixed(2)+"%"))
			return nil
		}

		if j.notifDispatcher != nil && changePercent.GreaterThanOrEqual(decimal.NewFromFloat(priceSurgeThreshold)) {
			direction := "surged"
			if price.LessThan(*asset.CurrentPrice) {
				direction = "dropped"
			}
			pct := changePercent.Mul(decimal.NewFromInt(100)).StringFixed(1)
			title := fmt.Sprintf("%s %s %s%%", asset.Ticker, direction, pct)
			msg := fmt.Sprintf("%s has %s by %s%% (from %s to %s).", asset.Ticker, direction, pct, asset.CurrentPrice.StringFixed(2), price.StringFixed(2))
			_ = j.notifDispatcher.Dispatch(asset.UserID, title, msg, models.NotificationTypeWarning)
		}
	}

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

func (j *AssetPriceSyncJob) updateTrades(tx *gorm.DB, assetID int64, price decimal.Decimal, now time.Time) error {
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
