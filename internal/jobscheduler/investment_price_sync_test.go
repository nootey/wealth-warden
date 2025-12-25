package jobscheduler_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/prices"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type InvestmentPriceSyncJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestInvestmentPriceSyncJobSuite(t *testing.T) {
	suite.Run(t, new(InvestmentPriceSyncJobTestSuite))
}

// Test that job runs
func (s *InvestmentPriceSyncJobTestSuite) TestInvestmentPriceSyncJob_Success() {
	logger := zaptest.NewLogger(s.T())

	client, err := prices.NewPriceFetchClient(s.TC.App.Config.FinanceAPIBaseURL)
	if err != nil {
		logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	job := jobscheduler.NewInvestmentPriceSyncJob(logger, s.TC.App, client)

	err = job.Run(s.Ctx)
	s.NoError(err)
}

// Tests that the job updates asset prices, values, P&L, and account balance non-cash flows
func (s *InvestmentPriceSyncJobTestSuite) TestInvestmentPriceSyncJob_UpdatesPricesAndBalances() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Manually set an old price to simulate price change
	oldPrice := decimal.NewFromInt(60000)
	err = s.TC.DB.Model(&models.InvestmentAsset{}).
		Where("id = ?", assetID).
		Updates(map[string]interface{}{
			"current_price": oldPrice,
			"current_value": oldPrice,
			"profit_loss":   decimal.NewFromInt(10000),
		}).Error
	s.Require().NoError(err)

	// Update balance to match the old price
	oldNonCashInflows := decimal.NewFromInt(10000)
	err = s.TC.DB.Model(&models.Balance{}).
		Where("account_id = ? AND as_of = ?", accID, today).
		Update("non_cash_inflows", oldNonCashInflows).Error
	s.Require().NoError(err)

	var balanceBefore models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBefore).Error
	s.Require().NoError(err)

	// Run price sync job
	logger := zaptest.NewLogger(s.T())
	client, err := prices.NewPriceFetchClient(s.TC.App.Config.FinanceAPIBaseURL)
	s.Require().NoError(err)

	job := jobscheduler.NewInvestmentPriceSyncJob(logger, s.TC.App, client)

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 30*time.Second)
	defer cancel3()

	err = job.Run(ctx3)
	if err != nil {
		if errors.Is(ctx3.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: job timed out")
		}
		s.Require().NoError(err)
	}

	// Verify asset updated
	var assetAfter models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfter).Error
	s.Require().NoError(err)

	s.Assert().NotNil(assetAfter.CurrentPrice, "price should be updated")
	s.Assert().NotNil(assetAfter.LastPriceUpdate, "last update should be set")
	s.Assert().True(assetAfter.CurrentValue.GreaterThan(decimal.Zero), "value should be updated")

	// Verify balance non-cash flows updated
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfter).Error
	s.Require().NoError(err)

	s.Assert().False(oldNonCashInflows.Equal(balanceAfter.NonCashInflows),
		"non-cash flows should be updated")
	s.Assert().True(balanceAfter.NonCashInflows.GreaterThan(oldNonCashInflows),
		"non-cash flows should have increased with price")

}
