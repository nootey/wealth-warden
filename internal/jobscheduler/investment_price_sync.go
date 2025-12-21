package jobscheduler

import (
	"context"
	"wealth-warden/internal/bootstrap"

	"github.com/Finnhub-Stock-API/finnhub-go"
	"go.uber.org/zap"
)

type InvestmentPriceSyncJob struct {
	logger    *zap.Logger
	container *bootstrap.Container
}

func NewInvestmentPriceSyncJob(logger *zap.Logger, container *bootstrap.Container) *InvestmentPriceSyncJob {
	return &InvestmentPriceSyncJob{
		logger:    logger,
		container: container,
	}
}

func (j *InvestmentPriceSyncJob) Run(ctx context.Context) error {
	j.logger.Info("Starting investment price sync job")

	// Create Finnhub client
	cfg := finnhub.NewConfiguration()
	cfg.AddDefaultHeader("X-Finnhub-Token", j.container.Config.FinnhubAPIKey)
	client := finnhub.NewAPIClient(cfg).DefaultApi

	// Fetch BTC price (Binance BTC/USDT)
	quote, _, err := client.Quote(ctx, "BINANCE:BTCUSDT")
	if err != nil {
		j.logger.Error("Failed to fetch BTC price", zap.Error(err))
		return err
	}

	j.logger.Info("BTC Price",
		zap.Float32("current", quote.C),
		zap.Float32("high", quote.H),
		zap.Float32("low", quote.L),
	)

	j.logger.Info("Investment price sync completed")
	return nil
}
