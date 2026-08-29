package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type InvestmentServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestInvestmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(InvestmentServiceTestSuite))
}

// assertTickerPriced checks that ticker_price_history holds a positive latest price for the ticker.
func (s *InvestmentServiceTestSuite) assertTickerPriced(ticker string) {
	s.T().Helper()
	var ph models.TickerPriceHistory
	err := s.TC.DB.WithContext(s.Ctx).
		Where("ticker = ?", ticker).
		Order("as_of DESC").
		First(&ph).Error
	s.Require().NoError(err)
	s.Assert().True(ph.Price.GreaterThan(decimal.Zero))
}

// Creates a stock asset with a valid ticker format (TICKER.EXCHANGE)
func (s *InvestmentServiceTestSuite) TestInsertAsset_ValidStockWithExchange() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(100000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromInt(10)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       qty,
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)
	s.Assert().Greater(assetID, int64(0))

	asset, err := svc.FetchInvestmentAssetByID(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	s.Assert().Equal("iShares Core MSCI World", asset.Name)
	s.Assert().Equal("IWDA.AS", asset.Ticker)
	s.Assert().Equal(models.InvestmentStock, asset.InvestmentType)
	s.Assert().True(qty.Equal(asset.Quantity))
	s.Assert().True(asset.AverageBuyPrice.Equal(decimal.Zero))
	s.Assert().Equal(accID, asset.AccountID)
	s.Assert().NotNil(asset.LastPriceUpdate)
	s.assertTickerPriced(asset.Ticker)
}

// Verifies that creating an asset seeds today's price into ticker_price_history
func (s *InvestmentServiceTestSuite) TestInsertAsset_SeedsPriceHistory() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(100000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	_, err = svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	today := time.Now().UTC().Truncate(24 * time.Hour)

	var ph models.TickerPriceHistory
	err = s.TC.DB.WithContext(s.Ctx).
		Where("ticker = ? AND as_of = ?", "BTC-USD", today).
		First(&ph).Error
	s.Require().NoError(err, "ticker_price_history should have a row for today after asset creation")
	s.Assert().True(ph.Price.GreaterThan(decimal.Zero), "seeded price should be > 0")
	s.Assert().NotEmpty(ph.Currency, "seeded price should have a currency")
}

// Verifies that a stock with an invalid/non-existent exchange returns an error
func (s *InvestmentServiceTestSuite) TestInsertAsset_StockWithInvalidExchange() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(100000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromInt(10)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "Apple Inc",
		Ticker:         "AAPL.INVALID",
		Quantity:       qty,
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)

	s.Require().Error(err)
	s.Assert().Equal(int64(0), assetID)

	var count int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentAsset{}).
		Where("user_id = ? AND ticker = ?", userID, "AAPL.INVALID").
		Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), count)
}

// Tests that crypto assets automatically append -USD if no currency is specified
func (s *InvestmentServiceTestSuite) TestInsertAsset_CryptoWithoutCurrency() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(100000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromFloat(0.5)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC", // Just ticker, no currency
		Quantity:       qty,
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)
	s.Assert().Greater(assetID, int64(0))

	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND user_id = ?", assetID, userID).
		First(&asset).Error
	s.Require().NoError(err)

	s.Assert().Equal("Bitcoin", asset.Name)
	s.Assert().Equal("BTC-USD", asset.Ticker) // Should auto-format to BTC-USD
	s.Assert().Equal(models.InvestmentCrypto, asset.InvestmentType)
	s.Assert().True(qty.Equal(asset.Quantity))
	s.assertTickerPriced(asset.Ticker)
}

// Tests inserting crypto with a specific currency pair (e.g., BTC-USDT)
func (s *InvestmentServiceTestSuite) TestInsertAsset_CryptoWithValidCurrency() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(100000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromFloat(1.5)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin brev",
		Ticker:         "BTC-EUR",
		Quantity:       qty,
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)
	s.Assert().Greater(assetID, int64(0))

	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND user_id = ?", assetID, userID).
		First(&asset).Error
	s.Require().NoError(err)

	s.Assert().Equal("Bitcoin brev", asset.Name)
	s.Assert().Equal("BTC-EUR", asset.Ticker)
	s.Assert().Equal(models.InvestmentCrypto, asset.InvestmentType)
	s.Assert().True(qty.Equal(asset.Quantity))
	s.assertTickerPriced(asset.Ticker)
}

// Tests that a buy trade correctly updates the asset's quantity and average buy price, and creates
// the necessary balance records with non-cash flows for unrealized P&L tracking
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_BuyUpdatesPriceAndQuantity() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	qty := decimal.NewFromInt(0)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Currency:       "EUR",
		Quantity:       qty,
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 1 BTC at 50k EUR today
	buyQty := decimal.NewFromInt(1)
	buyPrice := decimal.NewFromInt(50000)
	tradeReq := &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     buyQty,
		PricePerUnit: buyPrice,
		Currency:     "EUR",
	}

	tradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, tradeReq)
	s.Require().NoError(err)
	s.Assert().Greater(tradeID, int64(0))

	// Verify asset updated
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", assetID).
		First(&asset).Error
	s.Require().NoError(err)

	s.Assert().True(buyQty.Equal(asset.Quantity))
	s.Assert().True(buyPrice.Equal(asset.AverageBuyPrice))
	s.Assert().True(buyPrice.Equal(asset.ValueAtBuy))

	// Verify buy wrote cash_outflows
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)

	s.Assert().True(buyPrice.Equal(balance.CashOutflows),
		"buy should write cash_outflows of %s, got %s",
		buyPrice.String(), balance.CashOutflows.String())

	// Verify end balance = initial - purchase cost
	expectedEndBalance := initialBalance.Sub(buyPrice)
	s.Assert().True(expectedEndBalance.Equal(balance.EndBalance),
		"end balance should be %s, got %s",
		expectedEndBalance.String(), balance.EndBalance.String())

	// Verify snapshot reflects cash balance
	var snapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&snapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedEndBalance.Equal(snapshot.EndBalance),
		"snapshot should reflect cash balance of %s, got %s",
		expectedEndBalance.String(), snapshot.EndBalance.String())
}

// Tests that multiple buy trades correctly update the weighted average buy price and track unrealized P&L
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_MultipleBuysUpdateAverage() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 1 BTC at 50k
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Buy 0.5 BTC at 60k
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Buy 0.5 BTC at 55k
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(55000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify final asset state
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&asset).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(2).Equal(asset.Quantity),
		"total quantity should be 2, got %s", asset.Quantity.String())

	s.Assert().True(decimal.NewFromInt(53750).Equal(asset.AverageBuyPrice),
		"average buy price should be 53750, got %s", asset.AverageBuyPrice.String())

	s.Assert().True(decimal.NewFromInt(107500).Equal(asset.ValueAtBuy),
		"value at buy should be 107500, got %s", asset.ValueAtBuy.String())

	// Verify cash outflows = total spent (50k + 30k + 27.5k = 107.5k)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)

	expectedOutflows := decimal.NewFromInt(107500)
	s.Assert().True(expectedOutflows.Equal(balance.CashOutflows),
		"cash outflows should be %s, got %s",
		expectedOutflows.String(), balance.CashOutflows.String())

	// Verify end balance = 200k - 107.5k = 92.5k
	expectedEndBalance := initialBalance.Sub(expectedOutflows)
	s.Assert().True(expectedEndBalance.Equal(balance.EndBalance),
		"end balance should be %s, got %s",
		expectedEndBalance.String(), balance.EndBalance.String())
}

// Tests that selling an investment records realized gains/losses as cash inflows/outflows in the balance
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_SellRecordsRealizedPnL() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 2 BTC at 50k each = 100k total
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify balance after buy: 200k - 100k = 100k
	var balanceAfterBuy models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterBuy).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceAfterBuy.CashOutflows),
		"cash outflows after buy should be 100k, got %s", balanceAfterBuy.CashOutflows.String())

	// Sell 1 BTC at 90k (profit of 40k)
	sellQty := decimal.NewFromInt(1)
	sellPrice := decimal.NewFromInt(90000)

	sellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     sellQty,
		PricePerUnit: sellPrice,
		Currency:     "EUR",
	})
	s.Require().NoError(err)
	s.Assert().Greater(sellTradeID, int64(0))

	// Verify asset quantity reduced
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&asset).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(1).Equal(asset.Quantity))
	s.Assert().True(decimal.NewFromInt(50000).Equal(asset.AverageBuyPrice))

	// Proceeds = 1 * 90k = 90k (full proceeds recorded as cash_inflows)
	// Cost basis = 1 * 50k = 50k
	// Realized P&L = 40k (tracked on trade, not reflected in cash_inflows)
	expectedProceeds := decimal.NewFromInt(90000)

	var balanceAfterSell models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterSell).Error
	s.Require().NoError(err)

	s.Assert().True(expectedProceeds.Equal(balanceAfterSell.CashInflows),
		"full proceeds should be recorded as cash_inflows of %s, got %s",
		expectedProceeds.String(), balanceAfterSell.CashInflows.String())

	// End balance = 200k - 100k (buys) + 90k (full proceeds) = 190k
	expectedEndBalance := decimal.NewFromInt(190000)
	s.Assert().True(expectedEndBalance.Equal(balanceAfterSell.EndBalance),
		"end balance should be %s, got %s",
		expectedEndBalance.String(), balanceAfterSell.EndBalance.String())
}

// Tests that fees are correctly handled for both crypto (fee in tokens) and stocks/ETFs (fee in currency)
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_BuyWithFees() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	// Buy 1 BTC with 0.01 BTC fee = effective quantity 0.99 BTC
	cryptoAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	cryptoAssetID, err := svc.InsertAsset(s.Ctx, userID, cryptoAssetReq)
	s.Require().NoError(err)

	cryptoFee := decimal.NewFromFloat(0.01)
	cryptoTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Fee:          &cryptoFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)
	s.Assert().Greater(cryptoTradeID, int64(0))

	// Verify crypto asset: quantity = 0.99, value = 0.99 * 50k = 49,500
	var cryptoAsset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", cryptoAssetID).First(&cryptoAsset).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromFloat(0.99).Equal(cryptoAsset.Quantity),
		"crypto quantity should be 0.99, got %s", cryptoAsset.Quantity.String())
	s.Assert().True(decimal.NewFromFloat(49500).Equal(cryptoAsset.ValueAtBuy),
		"crypto value at buy should be 49500, got %s", cryptoAsset.ValueAtBuy.String())

	// Verify cash outflows = 49500 (effective value, crypto fee is in tokens not cash)
	var balanceAfterCrypto models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterCrypto).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromFloat(49500).Equal(balanceAfterCrypto.CashOutflows),
		"cash outflows should be 49500, got %s", balanceAfterCrypto.CashOutflows.String())

	// Buy 5 IWDA at 100 EUR with 3 EUR fee = value 500 EUR, cash out 503 EUR
	stockAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	stockAssetID, err := svc.InsertAsset(s.Ctx, userID, stockAssetReq)
	s.Require().NoError(err)

	stockFee := decimal.NewFromInt(3)
	stockTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &stockFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)
	s.Assert().Greater(stockTradeID, int64(0))

	// Verify stock asset: quantity = 5, value = 5 * 100 = 500 (pure trade value, fee separate)
	var stockAsset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", stockAssetID).First(&stockAsset).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(5).Equal(stockAsset.Quantity),
		"stock quantity should be 5, got %s", stockAsset.Quantity.String())
	s.Assert().True(decimal.NewFromInt(500).Equal(stockAsset.ValueAtBuy),
		"stock value at buy should be 500, got %s", stockAsset.ValueAtBuy.String())
	s.Assert().True(decimal.NewFromFloat(100).Equal(stockAsset.AverageBuyPrice),
		"stock avg buy price should be 100, got %s", stockAsset.AverageBuyPrice.String())

	// Verify total cash outflows = 49500 (crypto) + 503 (stock qty*price+fee) = 50003
	var finalBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&finalBalance).Error
	s.Require().NoError(err)

	expectedTotalOutflows := decimal.NewFromFloat(50003)
	s.Assert().True(expectedTotalOutflows.Equal(finalBalance.CashOutflows),
		"total cash outflows should be %s, got %s",
		expectedTotalOutflows.String(), finalBalance.CashOutflows.String())
}

// Tests that fees are correctly deducted from realized P&L when selling investments (both crypto and stocks)
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_SellWithFees() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	// Stock with fee (fee deducted from proceeds)
	stockAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	stockAssetID, err := svc.InsertAsset(s.Ctx, userID, stockAssetReq)
	s.Require().NoError(err)

	// Buy 10 shares at 100 EUR with 5 EUR fee = value 1005 (fee included in cost basis)
	buyFee := decimal.NewFromInt(5)
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &buyFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell 5 shares at 120 EUR with 3 EUR fee
	// Proceeds = (5 * 120) - 3 = 597
	// Cost basis = 5 * 100.5 = 502.5
	// Realized P&L = 597 - 502.5 = 94.5
	sellFee := decimal.NewFromInt(3)
	sellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(120),
		Fee:          &sellFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	var stockSellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", sellTradeID).First(&stockSellTrade).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(597).Equal(stockSellTrade.RealizedValue),
		"stock realized value should be 597 (proceeds after fee)")

	// Verify balance: buy wrote 1005 outflows (qty*price+fee), sell wrote 597 inflows
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)

	expectedOutflows := decimal.NewFromFloat(1005)
	s.Assert().True(expectedOutflows.Equal(balance.CashOutflows),
		"cash outflows should be %s, got %s", expectedOutflows.String(), balance.CashOutflows.String())

	expectedInflows := decimal.NewFromFloat(597)
	s.Assert().True(expectedInflows.Equal(balance.CashInflows),
		"cash inflows (full proceeds after fee) should be %s, got %s", expectedInflows.String(), balance.CashInflows.String())

	// Crypto with fee (fee in tokens)
	cryptoAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	cryptoAssetID, err := svc.InsertAsset(s.Ctx, userID, cryptoAssetReq)
	s.Require().NoError(err)

	// Buy 1 BTC with 0.01 BTC fee -> effective 0.99 BTC
	cryptoBuyFee := decimal.NewFromFloat(0.01)
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Fee:          &cryptoBuyFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell 0.5 BTC at 90k with 0.005 BTC fee
	// Full 0.5 BTC leaves holdings; the fee only reduces cash proceeds
	// Realized value = (0.5 - 0.005) * 90k = 0.495 * 90k = 44,550
	cryptoSellFee := decimal.NewFromFloat(0.005)
	cryptoSellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(90000),
		Fee:          &cryptoSellFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	var cryptoSellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", cryptoSellTradeID).First(&cryptoSellTrade).Error
	s.Require().NoError(err)

	expectedCryptoRealizedValue := decimal.NewFromFloat(0.495).Mul(decimal.NewFromInt(90000))
	s.Assert().True(expectedCryptoRealizedValue.Equal(cryptoSellTrade.RealizedValue),
		"crypto realized value should account for token fee: expected %s, got %s",
		expectedCryptoRealizedValue.String(), cryptoSellTrade.RealizedValue.String())

	var cryptoAssetAfterSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", cryptoAssetID).First(&cryptoAssetAfterSell).Error
	s.Require().NoError(err)

	// Bought 0.99, sold the full 0.5 (fee does not stay in the position): 0.99 - 0.5 = 0.49
	s.Assert().True(decimal.NewFromFloat(0.49).Equal(cryptoAssetAfterSell.Quantity),
		"remaining crypto quantity should be 0.49, got %s", cryptoAssetAfterSell.Quantity.String())
}

// Regression: selling a crypto position with a fee must remove the full sold quantity
// from holdings — the fee (in coin units) only reduces cash proceeds, it must not be
// left behind in the position.
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_SellWithFee_RemovesFullQuantity() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	})
	s.Require().NoError(err)

	// Buy 100 units at 1 EUR, no fee -> holdings = 100
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(100),
		PricePerUnit: decimal.NewFromInt(1),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell the entire 100 units at 1 EUR with a 2-unit fee
	sellFee := decimal.NewFromInt(2)
	sellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(100),
		PricePerUnit: decimal.NewFromInt(1),
		Fee:          &sellFee,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Holdings must be fully cleared — the 2 USDC fee is not left behind
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&asset).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(asset.Quantity),
		"holdings should be fully cleared after selling the entire position; the fee must not be left behind, got %s", asset.Quantity.String())

	// The sell trade records the full quantity sold (not quantity - fee)
	var sellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", sellTradeID).First(&sellTrade).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(100).Equal(sellTrade.Quantity),
		"sell trade quantity should be the full 100, got %s", sellTrade.Quantity.String())

	// Cash proceeds account for the fee: (100 - 2) * 1 = 98
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(98).Equal(balance.CashInflows),
		"cash inflows should be proceeds after fee (98), got %s", balance.CashInflows.String())
}

// Tests that deleting a sell trade reverses the realized P&L and recalculates the asset correctly
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentTrade_ReversesSellRealizedPnL() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 2 BTC at 50k EUR
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell 1 BTC at 90k EUR (40k profit)
	sellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify: balance should have 100k outflows (buy) and 90k inflows (full sell proceeds)
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceBeforeDelete.CashOutflows),
		"cash outflows should be 100k, got %s", balanceBeforeDelete.CashOutflows.String())
	s.Assert().True(decimal.NewFromInt(90000).Equal(balanceBeforeDelete.CashInflows),
		"cash inflows should be 90k, got %s", balanceBeforeDelete.CashInflows.String())

	// Delete the sell trade
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, sellTradeID)
	s.Require().NoError(err)

	// Verify asset restored to 2 BTC
	var assetAfterDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(2).Equal(assetAfterDelete.Quantity),
		"quantity should be restored to 2 BTC, got %s", assetAfterDelete.Quantity.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.AverageBuyPrice),
		"average buy price should be 50k, got %s", assetAfterDelete.AverageBuyPrice.String())

	// Verify realized P&L reversed — cash inflows back to 0
	var balanceAfterDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.Zero.Equal(balanceAfterDelete.CashInflows),
		"cash inflows should be 0 after reversing sell, got %s", balanceAfterDelete.CashInflows.String())

	// Cash outflows still 100k (buy not reversed)
	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceAfterDelete.CashOutflows),
		"cash outflows should remain 100k, got %s", balanceAfterDelete.CashOutflows.String())

	// Sell trade deleted
	var tradeCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("id = ?", sellTradeID).
		Count(&tradeCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), tradeCount, "sell trade should be deleted")
}

// Tests that deleting a buy trade recalculates the asset quantity and average price correctly
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentTrade_ReversesBuyAndRecalculates() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 1: 1 BTC at 50k EUR
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Buy 2: 1 BTC at 60k EUR
	secondBuyTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify: 2 BTC, avg 55k, outflows = 110k
	var assetBeforeDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(2).Equal(assetBeforeDelete.Quantity))
	s.Assert().True(decimal.NewFromInt(55000).Equal(assetBeforeDelete.AverageBuyPrice))

	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(110000).Equal(balanceBeforeDelete.CashOutflows),
		"cash outflows should be 110k before delete, got %s", balanceBeforeDelete.CashOutflows.String())

	// Delete second buy
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, secondBuyTradeID)
	s.Require().NoError(err)

	// Verify asset: 1 BTC at 50k
	var assetAfterDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(1).Equal(assetAfterDelete.Quantity),
		"quantity should be 1 BTC, got %s", assetAfterDelete.Quantity.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.AverageBuyPrice),
		"average price should be 50k, got %s", assetAfterDelete.AverageBuyPrice.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.ValueAtBuy),
		"value at buy should be 50k, got %s", assetAfterDelete.ValueAtBuy.String())

	// Verify cash outflows reversed: 110k - 60k = 50k
	var balanceAfterDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(50000).Equal(balanceAfterDelete.CashOutflows),
		"cash outflows should be 50k after reversing second buy, got %s", balanceAfterDelete.CashOutflows.String())

	// Trade deleted
	var tradeCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("id = ?", secondBuyTradeID).
		Count(&tradeCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), tradeCount, "buy trade should be deleted")

	var remainingTradeCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("asset_id = ?", assetID).
		Count(&remainingTradeCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), remainingTradeCount, "should have 1 trade remaining")
}

// Tests that deleting a buy trade is blocked if it would cause a sell trade to sell more than available quantity
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentTrade_BlockedByInsufficientQuantity() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 1: 1 BTC at 50k
	firstBuyTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Buy 2: 1 BTC at 60k
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell 1.5 BTC
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromFloat(1.5),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify 0.5 BTC remaining
	var assetBeforeDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromFloat(0.5).Equal(assetBeforeDelete.Quantity))

	// Try to delete first buy — would leave only 1 BTC but 1.5 was sold
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, firstBuyTradeID)
	s.Require().Error(err, "should not allow deleting buy that would make sell invalid")

	// Verify unchanged
	var assetAfterFailedDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterFailedDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromFloat(0.5).Equal(assetAfterFailedDelete.Quantity),
		"quantity should remain 0.5 BTC")

	var tradeCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("id = ?", firstBuyTradeID).
		Count(&tradeCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), tradeCount, "trade should still exist")
}

// Tests that deleting an asset deletes all associated trades and reverses all realized P&L
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentAsset_DeletesAllTradesAndReversesRealizedPnL() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	assetID, err := svc.InsertAsset(s.Ctx, userID, assetReq)
	s.Require().NoError(err)

	// Buy 2 BTC at 50k EUR each = 100k outflows
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Sell 1 BTC at 90k EUR = 90k full proceeds inflows
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Verify balance before delete: 100k outflows, 90k inflows (full proceeds), end = 190k
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceBeforeDelete.CashOutflows),
		"cash outflows should be 100k before delete")
	s.Assert().True(decimal.NewFromInt(90000).Equal(balanceBeforeDelete.CashInflows),
		"cash inflows should be 90k before delete")
	s.Assert().True(decimal.NewFromInt(190000).Equal(balanceBeforeDelete.EndBalance),
		"end balance should be 190k before delete")

	var tradeCountBefore int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("asset_id = ?", assetID).
		Count(&tradeCountBefore).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), tradeCountBefore, "should have 2 trades")

	// Delete the asset
	err = svc.DeleteInvestmentAsset(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	// Verify asset deleted
	var assetCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentAsset{}).
		Where("id = ?", assetID).
		Count(&assetCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), assetCount, "asset should be deleted")

	// Verify all trades deleted
	var tradeCountAfter int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("asset_id = ?", assetID).
		Count(&tradeCountAfter).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), tradeCountAfter, "all trades should be deleted")

	// Verify balance fully reversed: outflows and inflows both 0, end balance = initial
	var balanceAfterDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.Zero.Equal(balanceAfterDelete.CashOutflows),
		"cash outflows should be 0 after asset delete, got %s", balanceAfterDelete.CashOutflows.String())
	s.Assert().True(decimal.Zero.Equal(balanceAfterDelete.CashInflows),
		"cash inflows should be 0 after asset delete, got %s", balanceAfterDelete.CashInflows.String())
	s.Assert().True(initialBalance.Equal(balanceAfterDelete.EndBalance),
		"end balance should be restored to initial %s, got %s",
		initialBalance.String(), balanceAfterDelete.EndBalance.String())
}

// Tests that deleting an asset with no trades succeeds cleanly and leaves account state untouched
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentAsset_NoTrades_LeavesCleanState() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(50000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	err = svc.DeleteInvestmentAsset(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	var assetCount int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.InvestmentAsset{}).
		Where("id = ?", assetID).Count(&assetCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), assetCount, "asset should be deleted")

	// Balance and snapshot should be unaffected
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(balance.EndBalance),
		"end balance should be unchanged: expected %s, got %s",
		initialBalance.String(), balance.EndBalance.String())
}

// Tests that deleting a trade on a historical date recalculates snapshots from that date forward
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentTrade_HistoricalTrade_RecalculatesSnapshotsForward() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	tradDate := today.AddDate(0, 0, -3)
	initialBalance := decimal.NewFromInt(100000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      tradDate,
	})
	s.Require().NoError(err)

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	buyPrice := decimal.NewFromInt(40000)
	tradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      tradDate,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: buyPrice,
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Snapshot at trade date should reflect the outflow
	var snapBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, tradDate).
		First(&snapBefore).Error
	s.Require().NoError(err)
	expectedAfterBuy := initialBalance.Sub(buyPrice)
	s.Assert().True(expectedAfterBuy.Equal(snapBefore.EndBalance),
		"snapshot at trade date should reflect outflow: expected %s, got %s",
		expectedAfterBuy.String(), snapBefore.EndBalance.String())

	// Delete the trade
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, tradeID)
	s.Require().NoError(err)

	// Snapshot at trade date should now be back to initial balance
	var snapAtTrade models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, tradDate).
		First(&snapAtTrade).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapAtTrade.EndBalance),
		"snapshot at trade date should be restored: expected %s, got %s",
		initialBalance.String(), snapAtTrade.EndBalance.String())

	// Snapshot at today should also reflect the reversal (forward recalculation)
	var snapToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&snapToday).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapToday.EndBalance),
		"snapshot at today should also be recalculated: expected %s, got %s",
		initialBalance.String(), snapToday.EndBalance.String())
}

// Tests that a staking reward increases asset quantity but leaves average buy price (cost basis) unchanged.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_StakingIncreasesQuantityNotCostBasis() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	// Buy 1 BTC at 45000 EUR → avg_buy_price = 45000
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(45000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	stakingQty := decimal.NewFromFloat(0.5)
	incomeID, err := svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeStaking,
		Quantity:   &stakingQty,
		Currency:   "EUR",
	})
	s.Require().NoError(err)
	s.Assert().Greater(incomeID, int64(0))

	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&asset).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromFloat(1.5).Equal(asset.Quantity),
		"quantity should be buy qty 1 + staking 0.5 = 1.5, got %s", asset.Quantity.String())
	s.Assert().True(decimal.NewFromInt(45000).Equal(asset.AverageBuyPrice),
		"staking should not affect average buy price: expected 45000, got %s", asset.AverageBuyPrice.String())

	// Income FMV: 0.5 BTC * 45000 EUR (mock price for BTC-EUR)
	var income models.InvestmentIncome
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", incomeID).First(&income).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(22500).Equal(income.Amount),
		"staking income amount should be FMV 0.5*45000=22500, got %s", income.Amount.String())
}

// Tests that dividend income creates a linked is_system transaction with the net-of-tax amount.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_DividendCreatesLinkedIsSystemTransaction() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	amount := decimal.NewFromInt(50)
	taxWithheld := decimal.NewFromInt(10)
	incomeID, err := svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:     assetID,
		TxnDate:     today,
		IncomeType:  models.IncomeTypeDividend,
		Amount:      &amount,
		TaxWithheld: &taxWithheld,
		Currency:    "EUR",
	})
	s.Require().NoError(err)
	s.Assert().Greater(incomeID, int64(0))

	var income models.InvestmentIncome
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", incomeID).First(&income).Error
	s.Require().NoError(err)
	s.Require().NotNil(income.LinkedTransactionID, "dividend income should have a linked transaction ID")

	var txn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", *income.LinkedTransactionID).First(&txn).Error
	s.Require().NoError(err)

	// net amount = 50 - 10 tax withheld = 40 EUR
	s.Assert().True(decimal.NewFromInt(40).Equal(txn.Amount),
		"linked transaction amount should be 40 (50 gross - 10 tax withheld), got %s", txn.Amount.String())
	s.Assert().True(txn.IsSystem, "linked dividend transaction must be marked is_system=true")
	s.Assert().Equal(accID, txn.AccountID, "linked transaction should be in the asset's account")
	s.Assert().Equal("income", txn.TransactionType)
}

// Tests that deleting a staking income record reverses the quantity increment on the asset.
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentIncome_StakingReversesQuantity() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(45000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	stakingQty := decimal.NewFromFloat(0.5)
	incomeID, err := svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeStaking,
		Quantity:   &stakingQty,
		Currency:   "EUR",
	})
	s.Require().NoError(err)

	var assetBefore models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBefore).Error
	s.Require().NoError(err)
	s.Require().True(decimal.NewFromFloat(1.5).Equal(assetBefore.Quantity), "pre-condition: quantity should be 1.5")

	err = svc.DeleteInvestmentIncome(s.Ctx, userID, incomeID)
	s.Require().NoError(err)

	var assetAfter models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfter).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(1).Equal(assetAfter.Quantity),
		"quantity should revert to trade-only quantity of 1 after deleting staking, got %s", assetAfter.Quantity.String())
	s.Assert().True(decimal.NewFromInt(45000).Equal(assetAfter.AverageBuyPrice),
		"average buy price should remain 45000, got %s", assetAfter.AverageBuyPrice.String())

	var count int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.InvestmentIncome{}).Where("id = ?", incomeID).Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), count, "staking income record should be deleted")
}

// Tests that deleting a dividend income record cascades to the linked is_system transaction.
func (s *InvestmentServiceTestSuite) TestDeleteInvestmentIncome_DividendDeletesLinkedTransaction() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	amount := decimal.NewFromInt(100)
	incomeID, err := svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeDividend,
		Amount:     &amount,
		Currency:   "EUR",
	})
	s.Require().NoError(err)

	var income models.InvestmentIncome
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", incomeID).First(&income).Error
	s.Require().NoError(err)
	s.Require().NotNil(income.LinkedTransactionID)
	linkedTxnID := *income.LinkedTransactionID

	err = svc.DeleteInvestmentIncome(s.Ctx, userID, incomeID)
	s.Require().NoError(err)

	var incomeCount int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.InvestmentIncome{}).Where("id = ?", incomeID).Count(&incomeCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), incomeCount, "dividend income record should be deleted")

	// Transaction is soft-deleted (deleted_at set)
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.Transaction{}).
		Where("id = ? AND deleted_at IS NULL", linkedTxnID).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount, "linked is_system transaction should be soft-deleted")
}

// Tests that dividend transactions are excluded from analytics via the is_system=false filter.
// A non-system income transaction in the same account should be counted; the dividend should not.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_DividendExcludedFromAnalytics() {
	invSvc := s.TC.App.InvestmentService
	anaSvc := s.TC.App.AnalyticsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	year := now.Year()
	month := int(now.Month())

	initialBalance := decimal.NewFromInt(100000)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	assetID, err := invSvc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	// Find the seeded income category
	var incomeCat models.Category
	err = s.TC.DB.WithContext(s.Ctx).
		Where("classification = ? AND user_id IS NULL AND parent_id IS NULL", "income").
		First(&incomeCat).Error
	s.Require().NoError(err)

	// Insert a regular (non-system) income transaction of 200 EUR
	regularAmount := decimal.NewFromInt(200)
	desc := "Salary"
	err = s.TC.DB.WithContext(s.Ctx).Create(&models.Transaction{
		UserID:          userID,
		AccountID:       accID,
		CategoryID:      &incomeCat.ID,
		TransactionType: "income",
		Amount:          regularAmount,
		Currency:        "EUR",
		TxnDate:         today,
		Description:     &desc,
		IsSystem:        false,
	}).Error
	s.Require().NoError(err)

	// Record a dividend of 50 EUR → creates is_system=true transaction
	dividendAmount := decimal.NewFromInt(50)
	_, err = invSvc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeDividend,
		Amount:     &dividendAmount,
		Currency:   "EUR",
	})
	s.Require().NoError(err)

	stats, err := anaSvc.GetMonthlyStats(s.Ctx, userID, &accID, year, month)
	s.Require().NoError(err)
	s.Require().NotNil(stats)

	s.Assert().True(regularAmount.Equal(stats.Inflow),
		"only the non-system income (200) should appear in analytics; dividend (is_system=true) must be excluded, got inflow=%s",
		stats.Inflow.String())
}

// Tests that a dividend in a foreign currency is converted to the account's currency.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_DividendWithCurrencyConversion() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)
	// Account is in EUR
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account EUR",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	// Dividend of 110 USD; mock rate USD→EUR = 1/1.1 → expected linked txn = 100 EUR
	amount := decimal.NewFromInt(110)
	incomeID, err := svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeDividend,
		Amount:     &amount,
		Currency:   "USD",
	})
	s.Require().NoError(err)

	var income models.InvestmentIncome
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", incomeID).First(&income).Error
	s.Require().NoError(err)
	s.Require().NotNil(income.LinkedTransactionID)

	var txn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", *income.LinkedTransactionID).First(&txn).Error
	s.Require().NoError(err)

	// 110 USD × (1/1.1) = 100 EUR
	s.Assert().True(decimal.NewFromInt(100).Equal(txn.Amount),
		"110 USD dividend at mock rate 1/1.1 should convert to 100 EUR, got %s", txn.Amount.String())
	s.Assert().Equal("EUR", txn.Currency, "linked transaction currency should be the account currency")
}

// Tests that staking income creation fails when quantity is missing.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_Staking_MissingQuantityReturnsError() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	_, err = svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeStaking,
		Quantity:   nil,
		Currency:   "USD",
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "quantity")

	var count int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.InvestmentIncome{}).Where("asset_id = ?", assetID).Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), count, "no income record should be created on validation failure")
}

// Tests that dividend income creation fails when amount is missing.
func (s *InvestmentServiceTestSuite) TestCreateInvestmentIncome_Dividend_MissingAmountReturnsError() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	_, err = svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeDividend,
		Amount:     nil,
		Currency:   "EUR",
	})
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "amount")

	var count int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.InvestmentIncome{}).Where("asset_id = ?", assetID).Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), count, "no income record should be created on validation failure")
}

// A sell on an asset holding staking rewards must remove the same cost basis whether
// it goes through the incremental update or a later full recalculation, and the basis
// removed must match the cost basis the sell trade itself recorded.
func (s *InvestmentServiceTestSuite) TestSellWithStakingRewards_BasisAgreesAcrossPaths() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	// Buy 10 units at 100 EUR -> cost basis 1000, quantity 10
	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(100),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	// Staking reward of 2 units -> quantity 12, basis untouched at 1000
	stakedQty := decimal.NewFromInt(2)
	_, err = svc.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    today,
		IncomeType: models.IncomeTypeStaking,
		Quantity:   &stakedQty,
		Currency:   "EUR",
	})
	s.Require().NoError(err)

	var beforeSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&beforeSell).Error
	s.Require().NoError(err)
	s.Require().True(decimal.NewFromInt(12).Equal(beforeSell.Quantity),
		"staking reward should lift quantity to 12, got %s", beforeSell.Quantity.String())

	// Sell 6 of the 12 units held
	sellTradeID, err := svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(6),
		PricePerUnit: decimal.NewFromInt(120),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	var afterSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&afterSell).Error
	s.Require().NoError(err)

	var sellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", sellTradeID).First(&sellTrade).Error
	s.Require().NoError(err)

	basisRemovedIncrementally := beforeSell.ValueAtBuy.Sub(afterSell.ValueAtBuy)
	// The trade prices the sale off average_buy_price, stored at 4dp, so allow that quantisation
	tolerance := decimal.NewFromInt(6).Mul(decimal.NewFromFloat(0.0001))
	s.Assert().True(sellTrade.ValueAtBuy.Sub(basisRemovedIncrementally).Abs().LessThanOrEqual(tolerance),
		"sell trade recorded a cost basis of %s but %s left the asset",
		sellTrade.ValueAtBuy.String(), basisRemovedIncrementally.String())

	// Any later recalculation (price sync, income change, trade delete) must land on the
	// same numbers the incremental sell produced
	err = svc.RecalculateAssetPnL(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	var afterRecalc models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&afterRecalc).Error
	s.Require().NoError(err)

	s.Assert().True(afterSell.Quantity.Equal(afterRecalc.Quantity),
		"quantity drifted on recalc: %s -> %s", afterSell.Quantity.String(), afterRecalc.Quantity.String())
	s.Assert().True(afterSell.ValueAtBuy.Equal(afterRecalc.ValueAtBuy),
		"cost basis drifted on recalc: %s -> %s", afterSell.ValueAtBuy.String(), afterRecalc.ValueAtBuy.String())
	s.Assert().True(afterSell.AverageBuyPrice.Equal(afterRecalc.AverageBuyPrice),
		"average buy price drifted on recalc: %s -> %s", afterSell.AverageBuyPrice.String(), afterRecalc.AverageBuyPrice.String())
}

// A recalculation refreshes the ticker price, so it must write today's history row
// or the chart keeps replaying a stale price while the asset index moves on.
func (s *InvestmentServiceTestSuite) TestRecalculateAssetPnL_WritesTodaysPriceHistory() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
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

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Currency:       "EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(90),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	err = s.TC.DB.WithContext(s.Ctx).Model(&models.TickerPriceHistory{}).
		Where("ticker = ? AND as_of = ?", "IWDA.AS", today).
		Update("price", decimal.NewFromInt(42)).Error
	s.Require().NoError(err)

	err = svc.RecalculateAssetPnL(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	asset, err := svc.FetchInvestmentAssetByID(s.Ctx, userID, assetID)
	s.Require().NoError(err)

	var history models.TickerPriceHistory
	err = s.TC.DB.WithContext(s.Ctx).
		Where("ticker = ? AND as_of = ?", "IWDA.AS", today).
		First(&history).Error
	s.Require().NoError(err)

	s.Assert().False(history.Price.Equal(decimal.NewFromInt(42)),
		"recalc left today's history row at the stale forced price")
	s.Assert().True(asset.CurrentValue.Equal(history.Price.Mul(asset.Quantity)),
		"asset current_value %s does not match today's history price %s * qty %s",
		asset.CurrentValue.String(), history.Price.String(), asset.Quantity.String())
}

// Seeds a cached rate and removes it afterwards, so the shared exchange rate
// cache does not leak into other tests in the suite.
func (s *InvestmentServiceTestSuite) seedExchangeRate(from, to string, rate float64) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	err := s.TC.DB.WithContext(s.Ctx).Exec(`
		INSERT INTO exchange_rate_history (from_currency, to_currency, as_of, rate)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (from_currency, to_currency, as_of) DO UPDATE SET rate = EXCLUDED.rate
	`, from, to, today, rate).Error
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		s.TC.DB.Exec(`
			DELETE FROM exchange_rate_history
			WHERE from_currency = ? AND to_currency = ? AND as_of = ?
		`, from, to, today)
	})
}

func (s *InvestmentServiceTestSuite) newInvestmentAccount(userID int64, name string) int64 {
	balance := decimal.NewFromInt(1000000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          name,
		AccountTypeID: 5,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().Truncate(24 * time.Hour),
	})
	s.Require().NoError(err)
	return accID
}

// Buys quantity of ticker at price in EUR and returns the asset id
func (s *InvestmentServiceTestSuite) newHolding(userID, accID int64, ticker, name string, invType models.InvestmentType, quantity, price decimal.Decimal) int64 {
	svc := s.TC.App.InvestmentService
	today := time.Now().UTC().Truncate(24 * time.Hour)

	assetID, err := svc.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: invType,
		Name:           name,
		Ticker:         ticker,
		Currency:       "EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)

	if quantity.IsPositive() {
		_, err = svc.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
			AssetID:      assetID,
			TxnDate:      today,
			TradeType:    models.InvestmentBuy,
			Quantity:     quantity,
			PricePerUnit: price,
			Currency:     "EUR",
		})
		s.Require().NoError(err)
	}

	return assetID
}

func (s *InvestmentServiceTestSuite) allocationRow(rows []models.AllocationRow, key string) *models.AllocationRow {
	for i := range rows {
		if rows[i].Key == key {
			return &rows[i]
		}
	}
	return nil
}

// A holding priced in USD converts with the cached rate, and every group splits
// the same total four ways
func (s *InvestmentServiceTestSuite) TestFetchPortfolioAllocation_ConvertsAndWeights() {
	userID := int64(1)

	// The mock prices BTC-USD in USD, so this asset needs conversion to EUR
	s.seedExchangeRate("USD", "EUR", 0.9)

	accID := s.newInvestmentAccount(userID, "Investment Account")

	// 10 x IWDA.AS at 100 EUR -> 1000 EUR, no conversion
	s.newHolding(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(10), decimal.NewFromInt(100))

	// 0.02 x BTC-USD at 50000 USD -> 1000 USD -> 900 EUR
	s.newHolding(userID, accID, "BTC-USD", "Bitcoin", models.InvestmentCrypto,
		decimal.NewFromFloat(0.02), decimal.NewFromInt(1000))

	alloc, err := s.TC.App.InvestmentService.FetchPortfolioAllocation(s.Ctx, userID, "EUR")
	s.Require().NoError(err)

	s.Assert().Equal("EUR", alloc.Currency)
	s.Assert().Equal(0, alloc.UnpricedAssets)
	s.Assert().True(decimal.NewFromInt(1900).Equal(alloc.TotalValue),
		"total should be 1900 EUR, got %s", alloc.TotalValue.String())

	byCurrency := alloc.Groups["currency"]
	s.Require().Len(byCurrency, 2)
	usd := s.allocationRow(byCurrency, "USD")
	s.Require().NotNil(usd)
	s.Assert().True(decimal.NewFromInt(900).Equal(usd.Value),
		"USD holding should convert to 900 EUR, got %s", usd.Value.String())

	byType := alloc.Groups["type"]
	s.Require().Len(byType, 2)
	// Largest slice first
	s.Assert().Equal("stock", byType[0].Key)
	s.Assert().Equal("Stock", byType[0].Label)

	s.Require().Len(alloc.Groups["ticker"], 2)

	// One account holds everything, so its weight is the whole portfolio
	byAccount := alloc.Groups["account"]
	s.Require().Len(byAccount, 1)
	s.Assert().True(decimal.NewFromInt(1).Equal(byAccount[0].Weight))
	s.Assert().Equal("Investment Account", byAccount[0].Label)

	for name, rows := range alloc.Groups {
		sum := decimal.Zero
		for _, row := range rows {
			sum = sum.Add(row.Weight)
		}
		s.Assert().True(decimal.NewFromInt(1).Equal(sum),
			"weights in group %q should sum to 1, got %s", name, sum.String())
	}
}

// Closed accounts, empty positions and unpriced assets stay out of the total
func (s *InvestmentServiceTestSuite) TestFetchPortfolioAllocation_ExcludesUnpricedAndClosed() {
	userID := int64(1)

	accID := s.newInvestmentAccount(userID, "Open Account")
	closedAccID := s.newInvestmentAccount(userID, "Closed Account")

	s.newHolding(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(10), decimal.NewFromInt(100))

	// Held, but never priced
	s.newHolding(userID, accID, "BTC-USD", "Bitcoin", models.InvestmentCrypto,
		decimal.NewFromInt(1), decimal.NewFromInt(50000))
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		`DELETE FROM ticker_price_history WHERE ticker = ?`, "BTC-USD").Error)

	// Priced, but the position is empty
	s.newHolding(userID, accID, "BTC-EUR", "Bitcoin EUR", models.InvestmentCrypto,
		decimal.Zero, decimal.Zero)

	// Priced and held, but the account is closed
	s.newHolding(userID, closedAccID, "BTC-EUR", "Bitcoin EUR", models.InvestmentCrypto,
		decimal.NewFromInt(1), decimal.NewFromInt(45000))
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		`UPDATE accounts SET is_active = FALSE WHERE id = ?`, closedAccID).Error)

	alloc, err := s.TC.App.InvestmentService.FetchPortfolioAllocation(s.Ctx, userID, "EUR")
	s.Require().NoError(err)

	s.Assert().Equal(1, alloc.UnpricedAssets)
	s.Assert().True(decimal.NewFromInt(1000).Equal(alloc.TotalValue),
		"only the priced, open holding counts, got %s", alloc.TotalValue.String())

	tickers := alloc.Groups["ticker"]
	s.Require().Len(tickers, 1)
	s.Assert().Equal("IWDA.AS", tickers[0].Key)
}

// Two accounts hold the same ticker. A price sync must write exactly one
// ticker_price_history row for that ticker/day and still refresh every holder's value.
func (s *InvestmentServiceTestSuite) TestApplyTickerPrice_OneRowPerTickerManyHolders() {
	userID := int64(1)
	accA := s.newInvestmentAccount(userID, "Account A")
	accB := s.newInvestmentAccount(userID, "Account B")

	idA := s.newHolding(userID, accA, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(10), decimal.NewFromInt(100))
	idB := s.newHolding(userID, accB, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(4), decimal.NewFromInt(100))

	today := time.Now().UTC().Truncate(24 * time.Hour)

	changes, err := s.TC.App.InvestmentService.ApplyTickerPrice(
		s.Ctx, "IWDA.AS", decimal.NewFromInt(120), "EUR", time.Now().UTC())
	s.Require().NoError(err)
	s.Require().Len(changes, 2)

	var rowCount int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Model(&models.TickerPriceHistory{}).
		Where("ticker = ? AND as_of = ?", "IWDA.AS", today).Count(&rowCount).Error)
	s.Assert().Equal(int64(1), rowCount, "one ticker/day row for all holders")

	a, err := s.TC.App.InvestmentService.FetchInvestmentAssetByID(s.Ctx, userID, idA)
	s.Require().NoError(err)
	b, err := s.TC.App.InvestmentService.FetchInvestmentAssetByID(s.Ctx, userID, idB)
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(1200).Equal(a.CurrentValue), "holder A value = 10 * 120, got %s", a.CurrentValue.String())
	s.Assert().True(decimal.NewFromInt(480).Equal(b.CurrentValue), "holder B value = 4 * 120, got %s", b.CurrentValue.String())
}

// setLatestTickerPrice overwrites today's ticker_price_history row for a ticker.
func (s *InvestmentServiceTestSuite) setLatestTickerPrice(ticker string, price decimal.Decimal) {
	s.T().Helper()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Model(&models.TickerPriceHistory{}).
		Where("ticker = ? AND as_of = ?", ticker, today).
		Update("price", price).Error)
}

// The asset list derives current_value from the latest ticker price and can sort by it.
func (s *InvestmentServiceTestSuite) TestFetchInvestmentAssetsPaginated_DerivesValueAndSorts() {
	userID := int64(1)
	accID := s.newInvestmentAccount(userID, "Investment Account")

	small := s.newHolding(userID, accID, "BTC-EUR", "Bitcoin EUR", models.InvestmentCrypto,
		decimal.NewFromFloat(0.01), decimal.NewFromInt(45000)) // 0.01 * 45000 = 450
	big := s.newHolding(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(10), decimal.NewFromInt(100)) // 10 * 100 = 1000

	page := utils.PaginationParams{PageNumber: 1, RowsPerPage: 10, SortField: "current_value", SortOrder: "asc"}
	rows, _, err := s.TC.App.InvestmentService.FetchInvestmentAssetsPaginated(s.Ctx, userID, page, &accID)
	s.Require().NoError(err)
	s.Require().Len(rows, 2)

	s.Assert().Equal(small, rows[0].ID, "ascending: smaller current_value first")
	s.Assert().Equal(big, rows[1].ID)
	s.Assert().True(decimal.NewFromInt(450).Equal(rows[0].CurrentValue), "got %s", rows[0].CurrentValue.String())
	s.Assert().True(decimal.NewFromInt(1000).Equal(rows[1].CurrentValue), "got %s", rows[1].CurrentValue.String())

	page.SortOrder = "desc"
	rows, _, err = s.TC.App.InvestmentService.FetchInvestmentAssetsPaginated(s.Ctx, userID, page, &accID)
	s.Require().NoError(err)
	s.Assert().Equal(big, rows[0].ID, "descending: larger current_value first")
}

// A buy trade's PnL follows the latest ticker price; a sell trade's PnL is the
// realized figure and does not move when the price moves.
func (s *InvestmentServiceTestSuite) TestFetchInvestmentTradesPaginated_BuyTracksPriceSellIsFixed() {
	userID := int64(1)
	accID := s.newInvestmentAccount(userID, "Investment Account")
	today := time.Now().UTC().Truncate(24 * time.Hour)

	assetID := s.newHolding(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock,
		decimal.NewFromInt(10), decimal.NewFromInt(100)) // buy 10 @ 100, value_at_buy 1000

	_, err := s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(4),
		PricePerUnit: decimal.NewFromInt(120),
		Currency:     "EUR",
	}) // realized 480, basis removed 4*100 = 400, sell PnL = 80
	s.Require().NoError(err)

	s.setLatestTickerPrice("IWDA.AS", decimal.NewFromInt(150))

	page := utils.PaginationParams{PageNumber: 1, RowsPerPage: 10, SortField: "id", SortOrder: "asc"}
	rows, _, err := s.TC.App.InvestmentService.FetchInvestmentTradesPaginated(s.Ctx, userID, page, &assetID)
	s.Require().NoError(err)
	s.Require().Len(rows, 2)

	var buy, sell models.InvestmentTrade
	for _, r := range rows {
		if r.TradeType == models.InvestmentBuy {
			buy = r
		} else {
			sell = r
		}
	}

	// buy: 10 * 150 - 1000 = 500
	s.Assert().True(decimal.NewFromInt(500).Equal(buy.ProfitLoss), "buy PnL tracks price, got %s", buy.ProfitLoss.String())
	s.Assert().True(decimal.NewFromInt(1500).Equal(buy.CurrentValue), "buy value tracks price, got %s", buy.CurrentValue.String())
	// sell: realized 480 - basis 400 = 80, independent of the 150 price
	s.Assert().True(decimal.NewFromInt(80).Equal(sell.ProfitLoss), "sell PnL is the realized figure, got %s", sell.ProfitLoss.String())
}

// Creates a priced asset with no opening trade, so the caller can date its own
func (s *InvestmentServiceTestSuite) newAsset(userID, accID int64, ticker, name string, invType models.InvestmentType) int64 {
	assetID, err := s.TC.App.InvestmentService.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: invType,
		Name:           name,
		Ticker:         ticker,
		Currency:       "EUR",
		Quantity:       decimal.Zero,
	})
	s.Require().NoError(err)
	return assetID
}

// A double in one year is 100%/year. Cost-basis P&L says +100% either way.
func (s *InvestmentServiceTestSuite) TestFetchPortfolioReturns_AnnualisesOverOneYear() {
	userID := int64(1)
	accID := s.newInvestmentAccount(userID, "Investment Account")
	yearAgo := time.Now().UTC().Truncate(24*time.Hour).AddDate(-1, 0, 0)

	assetID := s.newAsset(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock)

	// 500 EUR out. The mock prices IWDA.AS at 100 EUR, so it holds 1000 EUR today.
	_, err := s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      yearAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(50),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	returns, err := s.TC.App.InvestmentService.FetchPortfolioReturns(s.Ctx, userID, "EUR")
	s.Require().NoError(err)

	s.Assert().Equal("EUR", returns.Currency)
	s.Require().NotNil(returns.Portfolio.Rate, "no rate: %s", returns.Portfolio.Reason)
	s.Assert().True(returns.Portfolio.Rate.Sub(decimal.NewFromInt(1)).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"a double in one year should be about 100%%/year, got %s", returns.Portfolio.Rate.String())

	s.Require().Len(returns.Assets, 1)
	s.Assert().Equal("IWDA.AS", returns.Assets[0].Key)
	s.Assert().True(decimal.NewFromInt(1000).Equal(returns.Assets[0].CurrentValue),
		"closing value should be 1000 EUR, got %s", returns.Assets[0].CurrentValue.String())
}

// Staking pays in tokens, already inside the closing value - never an inflow.
func (s *InvestmentServiceTestSuite) TestFetchPortfolioReturns_ExcludesStakingRewards() {
	userID := int64(1)
	accID := s.newInvestmentAccount(userID, "Investment Account")
	yearAgo := time.Now().UTC().Truncate(24*time.Hour).AddDate(-1, 0, 0)

	assetID := s.newAsset(userID, accID, "BTC-EUR", "Bitcoin", models.InvestmentCrypto)

	_, err := s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      yearAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(30000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	stakedQty := decimal.NewFromFloat(0.1)
	_, err = s.TC.App.InvestmentService.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
		AssetID:    assetID,
		TxnDate:    yearAgo,
		IncomeType: models.IncomeTypeStaking,
		Quantity:   &stakedQty,
		Currency:   "EUR",
	})
	s.Require().NoError(err)

	returns, err := s.TC.App.InvestmentService.FetchPortfolioReturns(s.Ctx, userID, "EUR")
	s.Require().NoError(err)

	s.Require().Len(returns.Assets, 1)
	s.Require().NotNil(returns.Assets[0].Rate, "no rate: %s", returns.Assets[0].Reason)

	// 1.1 BTC x 45000 = 49500 EUR closing against 30000 EUR paid, over one year
	s.Assert().True(decimal.NewFromInt(49500).Equal(returns.Assets[0].CurrentValue),
		"closing value should be 49500 EUR, got %s", returns.Assets[0].CurrentValue.String())
	s.Assert().True(returns.Assets[0].Rate.Sub(decimal.NewFromFloat(0.65)).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"rate should be about 65%%/year, got %s (counting the reward as cash gives about 80%%)",
		returns.Assets[0].Rate.String())
}

// A held asset with no price has buys but no closing value. Counting it reads
// the holding as a total loss and drags the portfolio rate down.
func (s *InvestmentServiceTestSuite) TestFetchPortfolioReturns_ExcludesUnpricedHoldings() {
	userID := int64(1)
	accID := s.newInvestmentAccount(userID, "Investment Account")
	yearAgo := time.Now().UTC().Truncate(24*time.Hour).AddDate(-1, 0, 0)

	pricedID := s.newAsset(userID, accID, "IWDA.AS", "iShares Core MSCI World", models.InvestmentStock)
	_, err := s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      pricedID,
		TxnDate:      yearAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(50),
		Currency:     "EUR",
	})
	s.Require().NoError(err)

	unpricedID := s.newAsset(userID, accID, "BTC-EUR", "Bitcoin", models.InvestmentCrypto)
	_, err = s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      unpricedID,
		TxnDate:      yearAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(30000),
		Currency:     "EUR",
	})
	s.Require().NoError(err)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		`DELETE FROM ticker_price_history WHERE ticker = ?`, "BTC-EUR").Error)

	returns, err := s.TC.App.InvestmentService.FetchPortfolioReturns(s.Ctx, userID, "EUR")
	s.Require().NoError(err)

	s.Assert().Equal(1, returns.UnpricedAssets)

	s.Require().Len(returns.Assets, 1)
	s.Assert().Equal("IWDA.AS", returns.Assets[0].Key)

	s.Require().NotNil(returns.Portfolio.Rate, "no rate: %s", returns.Portfolio.Reason)
	s.Assert().True(returns.Portfolio.Rate.Sub(decimal.NewFromInt(1)).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"the unpriced holding must not drag the portfolio rate down, got %s", returns.Portfolio.Rate.String())
	s.Assert().True(decimal.NewFromInt(1000).Equal(returns.Portfolio.CurrentValue),
		"only the priced holding counts, got %s", returns.Portfolio.CurrentValue.String())
}
