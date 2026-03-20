package jobscheduler

import (
	"context"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AssetPriceHistoryBackfillJob struct {
	logger           *zap.Logger
	investmentSvc    services.InvestmentServiceInterface
	db               *gorm.DB
	priceFetchClient finance.PriceFetcher
}

func NewAssetPriceHistoryBackfillJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	db *gorm.DB,
	priceFetchClient finance.PriceFetcher,
) *AssetPriceHistoryBackfillJob {
	return &AssetPriceHistoryBackfillJob{
		logger:           logger,
		investmentSvc:    investmentSvc,
		db:               db,
		priceFetchClient: priceFetchClient,
	}
}

func (j *AssetPriceHistoryBackfillJob) Run(ctx context.Context) error {
	// Find all assets with their earliest trade date
	type assetRow struct {
		ID             int64
		Ticker         string
		InvestmentType models.InvestmentType
		Currency       string
		EarliestTrade  time.Time
	}

	var assets []assetRow
	err := j.db.WithContext(ctx).Raw(`
		SELECT 
			ia.id,
			ia.ticker,
			ia.investment_type,
			ia.currency,
			MIN(it.txn_date) AS earliest_trade
		FROM investment_assets ia
		JOIN investment_trades it ON it.asset_id = ia.id
		JOIN accounts a ON a.id = ia.account_id
		WHERE a.is_active = TRUE AND a.closed_at IS NULL
		GROUP BY ia.id, ia.ticker, ia.investment_type, ia.currency
	`).Scan(&assets).Error
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		j.logger.Info("No assets to backfill price history for")
		return nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	totalInserted := 0
	totalSkipped := 0

	for _, asset := range assets {
		func() {
			assetCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			j.logger.Info("Backfilling price history",
				zap.Int64("asset_id", asset.ID),
				zap.String("ticker", asset.Ticker),
				zap.String("from", asset.EarliestTrade.Format("2006-01-02")),
				zap.String("to", today.Format("2006-01-02")),
			)

			inserted, skipped, err := j.backfillAsset(assetCtx, asset.ID, asset.Ticker, asset.InvestmentType, asset.EarliestTrade, today)
			if err != nil {
				j.logger.Error("Failed to backfill asset",
					zap.Int64("asset_id", asset.ID),
					zap.String("ticker", asset.Ticker),
					zap.String("error", err.Error()),
				)
				return
			}

			totalInserted += inserted
			totalSkipped += skipped

			j.logger.Info("Asset backfill complete",
				zap.Int64("asset_id", asset.ID),
				zap.String("ticker", asset.Ticker),
				zap.Int("inserted", inserted),
				zap.Int("skipped", skipped),
			)
		}()
	}

	j.logger.Info("Price history backfill completed",
		zap.Int("total_inserted", totalInserted),
		zap.Int("total_skipped", totalSkipped),
	)

	return nil
}

func (j *AssetPriceHistoryBackfillJob) backfillAsset(ctx context.Context, assetID int64, ticker string, investmentType models.InvestmentType, from, to time.Time) (inserted, skipped int, err error) {
	from = from.UTC().Truncate(24 * time.Hour)
	to = to.UTC().Truncate(24 * time.Hour)

	// Find dates that already have price history so we can skip them
	var existingDates []time.Time
	err = j.db.WithContext(ctx).Raw(`
		SELECT as_of FROM asset_price_history
		WHERE asset_id = ? AND as_of >= ? AND as_of <= ?
	`, assetID, from, to).Scan(&existingDates).Error
	if err != nil {
		return 0, 0, err
	}

	existingSet := make(map[string]bool, len(existingDates))
	for _, d := range existingDates {
		existingSet[d.UTC().Truncate(24*time.Hour).Format("2006-01-02")] = true
	}

	current := from
	requestCount := 0

	for !current.After(to) {
		dateKey := current.Format("2006-01-02")

		// Skip weekends — markets closed, no price data
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			current = current.AddDate(0, 0, 1)
			continue
		}

		// Skip if already exists
		if existingSet[dateKey] {
			skipped++
			current = current.AddDate(0, 0, 1)
			continue
		}

		// Rate limit — pause every 5 requests
		if requestCount > 0 && requestCount%10 == 0 {
			select {
			case <-ctx.Done():
				return inserted, skipped, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}

		priceData, err := j.priceFetchClient.GetAssetPriceOnDate(ctx, ticker, investmentType, current)
		if err != nil {
			j.logger.Info("Failed to fetch historical price",
				zap.String("ticker", ticker),
				zap.String("date", dateKey),
				zap.String("error", err.Error()),
			)
			current = current.AddDate(0, 0, 1)
			requestCount++
			continue
		}

		price := decimal.NewFromFloat(priceData.Price)
		if price.IsZero() || price.IsNegative() {
			j.logger.Info("Invalid price received, skipping",
				zap.String("ticker", ticker),
				zap.String("date", dateKey),
			)
			current = current.AddDate(0, 0, 1)
			requestCount++
			continue
		}

		if err := j.investmentSvc.UpsertAssetPrice(ctx, nil, assetID, current, price, priceData.Currency); err != nil {
			j.logger.Info("Failed to insert price history",
				zap.Int64("asset_id", assetID),
				zap.String("date", dateKey),
				zap.String("error", err.Error()))
		} else {
			inserted++
		}

		requestCount++
		current = current.AddDate(0, 0, 1)
	}

	return inserted, skipped, nil
}
