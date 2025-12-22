package jobscheduler

import (
	"context"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/prices"

	"go.uber.org/zap"
)

type InvestmentPriceSyncJob struct {
	logger           *zap.Logger
	container        *bootstrap.Container
	priceFetchClient prices.PriceFetcher
}

func NewInvestmentPriceSyncJob(logger *zap.Logger, container *bootstrap.Container, priceFetchClient prices.PriceFetcher) *InvestmentPriceSyncJob {
	return &InvestmentPriceSyncJob{
		logger:           logger,
		container:        container,
		priceFetchClient: priceFetchClient,
	}
}

func (j *InvestmentPriceSyncJob) Run(ctx context.Context) error {
	j.logger.Info("Starting investment price sync job")

	if j.priceFetchClient != nil {
		btc, err := j.priceFetchClient.GetAssetPrice(ctx, "btc", "crypto")
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintln("BTC price info", btc))
	}
	
	j.logger.Info("Investment price sync completed")
	return nil
}
