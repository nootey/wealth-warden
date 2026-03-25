package jobscheduler

import (
	"context"
	"sync"
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

	type result struct {
		asset    assetRow
		inserted int
		skipped  int
		err      error
	}

	jobs := make(chan assetRow, len(assets))
	results := make(chan result, len(assets))

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for asset := range jobs {
				assetCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				inserted, skipped, err := j.backfillAsset(assetCtx, asset.ID, asset.Ticker, asset.InvestmentType, asset.EarliestTrade, today)
				cancel()
				results <- result{asset: asset, inserted: inserted, skipped: skipped, err: err}
			}
		}()
	}

	for _, asset := range assets {
		jobs <- asset
	}
	close(jobs)

	wg.Wait()
	close(results)

	totalInserted := 0
	totalSkipped := 0
	for r := range results {
		if r.err != nil {
			j.logger.Error("Failed to backfill asset",
				zap.Int64("asset_id", r.asset.ID),
				zap.String("ticker", r.asset.Ticker),
				zap.Error(r.err),
			)
			continue
		}
		j.logger.Info("Asset backfill complete",
			zap.Int64("asset_id", r.asset.ID),
			zap.String("ticker", r.asset.Ticker),
			zap.Int("inserted", r.inserted),
			zap.Int("skipped", r.skipped),
		)
		totalInserted += r.inserted
		totalSkipped += r.skipped
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
	var batch []models.AssetPriceHistory

	for !current.After(to) {
		dateKey := current.Format("2006-01-02")

		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			current = current.AddDate(0, 0, 1)
			continue
		}

		if existingSet[dateKey] {
			skipped++
			current = current.AddDate(0, 0, 1)
			continue
		}

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

		batch = append(batch, models.AssetPriceHistory{
			AssetID:  assetID,
			AsOf:     current,
			Price:    price,
			Currency: priceData.Currency,
		})

		requestCount++
		current = current.AddDate(0, 0, 1)
	}

	if len(batch) > 0 {
		if err := j.investmentSvc.UpsertAssetPrice(ctx, nil, batch); err != nil {
			j.logger.Error("Failed to insert price history batch",
				zap.Int64("asset_id", assetID),
				zap.Int("count", len(batch)),
				zap.Error(err))
			return 0, skipped, err
		}
		inserted = len(batch)
	}

	return inserted, skipped, nil
}
