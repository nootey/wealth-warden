package jobs

import (
	"context"
	"fmt"
	"github.com/riverqueue/river"
	"sync"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const priceSurgeThreshold = 0.10

type AssetPriceSyncWorker struct {
	river.WorkerDefaults[jobqueue.AssetPriceSyncArgs]
	logger *zap.Logger
	job    *AssetPriceSyncJob
}

func NewAssetPriceSyncWorker(logger *zap.Logger, job *AssetPriceSyncJob) *AssetPriceSyncWorker {
	return &AssetPriceSyncWorker{logger: logger, job: job}
}

// Fetching paces itself at one ticker per second, so the ceiling has to clear
// the whole ticker list plus the writes that follow.
func (w *AssetPriceSyncWorker) Timeout(*river.Job[jobqueue.AssetPriceSyncArgs]) time.Duration {
	return 30 * time.Minute
}

func (w *AssetPriceSyncWorker) Work(ctx context.Context, _ *river.Job[jobqueue.AssetPriceSyncArgs]) error {
	w.logger.Info("Starting asset price sync ...")
	if err := w.job.Run(ctx); err != nil {
		w.logger.Error("Price sync failed", zap.Error(err))
		return err
	}
	w.logger.Info("Price sync completed")
	return nil
}

type AssetPriceSyncJob struct {
	logger            *zap.Logger
	investmentSvc     services.InvestmentServiceInterface
	priceFetchClient  finance.PriceFetcher
	notifDispatcher   jobqueue.NotificationDispatcher
	concurrentWorkers int
}

func NewAssetPriceSyncJob(
	logger *zap.Logger,
	investmentSvc services.InvestmentServiceInterface,
	priceFetchClient finance.PriceFetcher,
	notifDispatcher jobqueue.NotificationDispatcher,
	concurrentWorkers int,
) *AssetPriceSyncJob {
	if concurrentWorkers < 1 {
		concurrentWorkers = defaultWorkers
	}
	return &AssetPriceSyncJob{
		logger:            logger,
		investmentSvc:     investmentSvc,
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

	pairs, err := j.investmentSvc.GetActiveCurrencyPairs(ctx)
	if err != nil {
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

	userIDs, err := j.investmentSvc.GetUserIDsWithActiveInvestments(ctx)
	if err != nil {
		return err
	}

	var (
		mu     sync.Mutex
		failed int
	)

	g := new(errgroup.Group)
	g.SetLimit(j.concurrentWorkers)

	for _, userID := range userIDs {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			if err := j.investmentSvc.UpdateSnapshotMarketValues(ctx, userID); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				return fmt.Errorf("user %d: %w", userID, err)
			}
			return nil
		})
	}
	firstErr := g.Wait()

	j.logger.Info("Snapshot market values refreshed",
		zap.Int("total", len(userIDs)),
		zap.Int("failed", failed))

	if firstErr != nil {
		j.logger.Error("Snapshot market value refresh had failures",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	return ctx.Err()
}

func (j *AssetPriceSyncJob) getAssetsToUpdate(ctx context.Context) ([]models.AssetPriceSyncRow, error) {
	assets, err := j.investmentSvc.GetTickersForPriceSync(ctx)
	if err != nil {
		j.logger.Error("Failed to fetch assets", zap.Error(err))
		return nil, err
	}

	j.logger.Info("Found assets to update", zap.Int("count", len(assets)))
	return assets, nil
}

func (j *AssetPriceSyncJob) fetchPrices(ctx context.Context, assets []models.AssetPriceSyncRow) (map[string]*finance.PriceData, error) {

	priceData := make(map[string]*finance.PriceData)
	start := time.Now()

	for i, asset := range assets {
		// Add delay between requests to avoid rate limiting
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}

		price, err := j.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err != nil {
			j.logger.Warn("Failed to fetch price",
				zap.String("ticker", asset.Ticker),
				zap.Error(err))
			continue
		}

		if price == nil {
			j.logger.Error("No price received", zap.String("ticker", asset.Ticker))
			continue
		}

		if price.Price <= 0 {
			j.logger.Error("Invalid price received",
				zap.String("ticker", asset.Ticker),
				zap.Float64("price", price.Price))
			continue
		}

		priceData[asset.Ticker] = price
	}

	j.logger.Info("Prices fetched",
		zap.Int("successful", len(priceData)),
		zap.Int("tickers", len(assets)),
		zap.Duration("took", time.Since(start)))

	failedCount := len(assets) - len(priceData)
	if failedCount > 0 {
		j.logger.Warn("Some prices failed to fetch",
			zap.Int("failed_count", failedCount),
			zap.Int("total_assets", len(assets)))
	}

	return priceData, nil
}

func (j *AssetPriceSyncJob) updateAssetsAndTrades(ctx context.Context, priceData map[string]*finance.PriceData) (int, error) {
	now := time.Now()

	updatedCount := 0
	failed := 0

	for ticker, price := range priceData {
		if ctx.Err() != nil {
			return updatedCount, ctx.Err()
		}

		changes, err := j.investmentSvc.ApplyTickerPrice(ctx, ticker, decimal.NewFromFloat(price.Price), price.Currency, now)
		if err != nil {
			failed++
			j.logger.Error("Failed to update assets for ticker",
				zap.String("ticker", ticker),
				zap.Error(err))
			continue
		}

		updatedCount += len(changes)
		j.notifyPriceSurges(ctx, changes)
	}

	if failed > 0 {
		j.logger.Warn("Some tickers failed to update",
			zap.Int("failed", failed),
			zap.Int("total", len(priceData)))
	}

	return updatedCount, nil
}

// The prices are already written when this runs, so a cancelled ctx must not
// swallow the notifications for them.
func (j *AssetPriceSyncJob) notifyPriceSurges(ctx context.Context, changes []models.AssetPriceChange) {
	if j.notifDispatcher == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)

	for _, c := range changes {
		if c.OldPrice == nil || c.OldPrice.IsZero() {
			continue
		}

		changePercent := c.NewPrice.Sub(*c.OldPrice).Div(*c.OldPrice).Abs()
		if changePercent.LessThan(decimal.NewFromFloat(priceSurgeThreshold)) {
			continue
		}

		direction := "surged"
		if c.NewPrice.LessThan(*c.OldPrice) {
			direction = "dropped"
		}
		pct := changePercent.Mul(decimal.NewFromInt(100)).StringFixed(1)
		title := fmt.Sprintf("%s %s %s%%", c.Ticker, direction, pct)
		msg := fmt.Sprintf("%s has %s by %s%% (from %s to %s).", c.Ticker, direction, pct, c.OldPrice.StringFixed(2), c.NewPrice.StringFixed(2))
		if err := j.notifDispatcher.Dispatch(ctx, c.UserID, title, msg, models.NotificationTypeWarning); err != nil {
			j.logger.Error("Failed to dispatch notification",
				zap.Int64("user_id", c.UserID),
				zap.String("title", title),
				zap.Error(err))
		}
	}
}
