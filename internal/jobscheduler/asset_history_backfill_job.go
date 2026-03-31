package jobscheduler

import (
	"context"
	"sync"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AssetPriceHistoryBackfillJob struct {
	logger        *zap.Logger
	investmentSvc services.InvestmentServiceInterface
	db            *gorm.DB
}

func NewAssetPriceHistoryBackfillJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	db *gorm.DB,
) *AssetPriceHistoryBackfillJob {
	return &AssetPriceHistoryBackfillJob{
		logger:        logger,
		investmentSvc: investmentSvc,
		db:            db,
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

	jobs := make(chan assetRow, len(assets))
	errs := make(chan error, len(assets))

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for asset := range jobs {
				assetCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				err := j.investmentSvc.BackfillAssetPriceHistory(assetCtx, asset.ID, asset.Ticker, asset.InvestmentType, asset.EarliestTrade, today)
				cancel()
				if err != nil {
					j.logger.Error("Failed to backfill asset",
						zap.Int64("asset_id", asset.ID),
						zap.String("ticker", asset.Ticker),
						zap.Error(err),
					)
					errs <- err
				} else {
					j.logger.Info("Asset backfill complete",
						zap.Int64("asset_id", asset.ID),
						zap.String("ticker", asset.Ticker),
					)
					errs <- nil
				}
			}
		}()
	}

	for _, asset := range assets {
		jobs <- asset
	}
	close(jobs)

	wg.Wait()
	close(errs)

	failed := 0
	for err := range errs {
		if err != nil {
			failed++
		}
	}

	j.logger.Info("Price history backfill completed",
		zap.Int("total", len(assets)),
		zap.Int("failed", failed),
	)

	return nil
}
