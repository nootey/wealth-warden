package jobscheduler

import (
	"context"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/prices"

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

	// Create price fetch client
	client, err := prices.NewPriceFetchClient(j.container.Config.FinanceAPIBaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize price client: %w", err)
	}

	btc, err := client.GetAssetPrice(ctx, "btc", "crypto")
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintln("BTC price info", btc))

	j.logger.Info("Investment price sync completed")
	return nil
}
