package jobs_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type AssetPriceSyncJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAssetPriceSyncJobSuite(t *testing.T) {
	suite.Run(t, new(AssetPriceSyncJobTestSuite))
}

// setTickerPrice overwrites the newest ticker_price_history row for a ticker.
func (s *AssetPriceSyncJobTestSuite) setTickerPrice(ticker string, price decimal.Decimal) {
	s.T().Helper()
	err := s.TC.DB.WithContext(s.Ctx).Model(&models.TickerPriceHistory{}).
		Where("ticker = ?", ticker).
		Update("price", price).Error
	s.Require().NoError(err)
}

// latestTickerPrice returns the newest ticker_price_history price for a ticker.
func (s *AssetPriceSyncJobTestSuite) latestTickerPrice(ticker string) decimal.Decimal {
	s.T().Helper()
	var ph models.TickerPriceHistory
	err := s.TC.DB.WithContext(s.Ctx).
		Where("ticker = ?", ticker).
		Order("as_of DESC").
		First(&ph).Error
	s.Require().NoError(err)
	return ph.Price
}

// Test that job runs with no assets
func (s *AssetPriceSyncJobTestSuite) TestAssetPriceSyncJob_Success() {
	logger := zaptest.NewLogger(s.T())
	job := jobs.NewAssetPriceSyncJob(logger, s.TC.App.InvestmentService, &tests.MockPriceFetcher{}, nil, 0)

	err := job.Run(s.Ctx)
	s.NoError(err)
}

// Tests that an asset whose new price is >90% below the current price is skipped to prevent data corruption
func (s *AssetPriceSyncJobTestSuite) TestAssetPriceSyncJob_SkipsExtremePriceDrop() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	_, err = invSvc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	// Mock returns BTC-USD at 50,000 — set the last known ticker price to 1,000,000 so the "new" price is a >90% drop
	inflatedPrice := decimal.NewFromInt(1000000)
	s.setTickerPrice("BTC-USD", inflatedPrice)

	logger := zaptest.NewLogger(s.T())
	job := jobs.NewAssetPriceSyncJob(logger, s.TC.App.InvestmentService, &tests.MockPriceFetcher{}, nil, 0)

	err = job.Run(s.Ctx)
	s.Require().NoError(err)

	s.Assert().True(inflatedPrice.Equal(s.latestTickerPrice("BTC-USD")),
		"ticker price should not have been updated on extreme drop: expected %s, got %s",
		inflatedPrice.String(), s.latestTickerPrice("BTC-USD").String())
}

// Tests that the job updates asset prices, values, P&L, and account balance non-cash flows
func (s *AssetPriceSyncJobTestSuite) TestAssetPriceSyncJob_UpdatesPricesAndBalances() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
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

	// Manually set an old ticker price to simulate a price change on the next sync
	oldPrice := decimal.NewFromInt(60000)
	s.setTickerPrice("BTC-USD", oldPrice)

	// Run price sync job
	logger := zaptest.NewLogger(s.T())
	job := jobs.NewAssetPriceSyncJob(logger, s.TC.App.InvestmentService, &tests.MockPriceFetcher{}, nil, 0)

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 30*time.Second)
	defer cancel3()

	err = job.Run(ctx3)
	if err != nil {
		if errors.Is(ctx3.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: job timed out")
		}
		s.Require().NoError(err)
	}

	// Verify asset value updated (value and PnL are derived from the latest ticker price)
	assetAfter, err := s.TC.App.InvestmentService.FetchInvestmentAssetByID(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	s.Assert().NotNil(assetAfter.LastPriceUpdate, "last update should be set")
	s.Assert().True(assetAfter.CurrentValue.GreaterThan(decimal.Zero), "current value should be updated")

	// Verify ticker price history refreshed away from the stale old price
	var priceHistory models.TickerPriceHistory
	err = s.TC.DB.WithContext(s.Ctx).
		Where("ticker = ? AND as_of = ?", "BTC-USD", today).
		First(&priceHistory).Error
	s.Require().NoError(err)
	s.Assert().True(priceHistory.Price.GreaterThan(decimal.Zero), "price history should be recorded")
	s.Assert().False(oldPrice.Equal(priceHistory.Price), "price sync should have replaced the stale price")

	// Verify cash balance reduced by purchase cost (buy wrote cash_outflows)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(balance.CashOutflows.GreaterThan(decimal.Zero), "buy should have written cash outflows")
}

type tickerFailingInvestmentService struct {
	services.InvestmentServiceInterface
	failTicker string
}

func (f *tickerFailingInvestmentService) ApplyTickerPrice(ctx context.Context, ticker string, price decimal.Decimal, currency string, now time.Time) ([]models.AssetPriceChange, error) {
	if ticker == f.failTicker {
		return nil, errors.New("simulated database failure")
	}
	return f.InvestmentServiceInterface.ApplyTickerPrice(ctx, ticker, price, currency, now)
}

// Tests that one failing ticker does not roll back the tickers that already succeeded
func (s *AssetPriceSyncJobTestSuite) TestAssetPriceSyncJob_FailedTickerDoesNotRollBackOthers() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	_, err = invSvc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	_, err = invSvc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentETF,
		Name:           "iShares MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	// Prices close to the mock's so the extreme-drop guard does not skip either asset
	s.setTickerPrice("BTC-USD", decimal.NewFromInt(45000))
	s.setTickerPrice("IWDA.AS", decimal.NewFromInt(90))

	failingSvc := &tickerFailingInvestmentService{
		InvestmentServiceInterface: invSvc,
		failTicker:                 "BTC-USD",
	}

	logger := zaptest.NewLogger(s.T())
	job := jobs.NewAssetPriceSyncJob(logger, failingSvc, &tests.MockPriceFetcher{}, nil, 0)

	err = job.Run(s.Ctx)
	s.Require().NoError(err, "one bad ticker must not fail the whole run")

	s.Assert().True(decimal.NewFromInt(100).Equal(s.latestTickerPrice("IWDA.AS")),
		"the healthy ticker must stay committed: expected 100, got %s", s.latestTickerPrice("IWDA.AS").String())

	s.Assert().True(decimal.NewFromInt(45000).Equal(s.latestTickerPrice("BTC-USD")),
		"the failed ticker must stay at its old price: expected 45000, got %s", s.latestTickerPrice("BTC-USD").String())
}
