package services_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type InvestmentServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestInvestmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(InvestmentServiceTestSuite))
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out (network issue)")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(assetID, int64(0))

	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND user_id = ?", assetID, userID).
		First(&asset).Error
	s.Require().NoError(err)

	s.Assert().Equal("iShares Core MSCI World", asset.Name)
	s.Assert().Equal("IWDA.AS", asset.Ticker)
	s.Assert().Equal(models.InvestmentStock, asset.InvestmentType)
	s.Assert().True(qty.Equal(asset.Quantity))
	s.Assert().True(asset.AverageBuyPrice.Equal(decimal.Zero))
	s.Assert().Equal(accID, asset.AccountID)
	s.Assert().NotNil(asset.CurrentPrice)
	s.Assert().NotNil(asset.LastPriceUpdate)
	s.Assert().True(asset.CurrentPrice.GreaterThan(decimal.Zero))
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)

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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
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
	s.Assert().NotNil(asset.CurrentPrice)
	s.Assert().True(asset.CurrentPrice.GreaterThan(decimal.Zero))
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
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
	s.Assert().NotNil(asset.CurrentPrice)
	s.Assert().True(asset.CurrentPrice.GreaterThan(decimal.Zero))
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	tradeID, err := svc.InsertInvestmentTrade(ctx2, userID, tradeReq)
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 1 BTC at 50k
	ctx1, cancel1 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel1()

	_, err = svc.InsertInvestmentTrade(ctx1, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx1.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 0.5 BTC at 60k
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 0.5 BTC at 55k
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	_, err = svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(55000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 2 BTC at 50k each = 100k total
	ctx1, cancel1 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel1()

	_, err = svc.InsertInvestmentTrade(ctx1, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx1.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	sellTradeID, err := svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     sellQty,
		PricePerUnit: sellPrice,
		Currency:     "EUR",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(sellTradeID, int64(0))

	// Verify asset quantity reduced
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&asset).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(1).Equal(asset.Quantity))
	s.Assert().True(decimal.NewFromInt(50000).Equal(asset.AverageBuyPrice))

	// Proceeds = 1 * 90k = 90k
	// Cost basis = 1 * 50k = 50k
	// Realized P&L = 40k
	expectedRealizedPnL := decimal.NewFromInt(40000)

	var balanceAfterSell models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterSell).Error
	s.Require().NoError(err)

	s.Assert().True(expectedRealizedPnL.Equal(balanceAfterSell.CashInflows),
		"realized P&L should be recorded as cash_inflows of %s, got %s",
		expectedRealizedPnL.String(), balanceAfterSell.CashInflows.String())

	// End balance = 200k - 100k (buys) + 40k (realized P&L) = 140k
	expectedEndBalance := decimal.NewFromInt(140000)
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	cryptoAssetID, err := svc.InsertAsset(ctx, userID, cryptoAssetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	cryptoFee := decimal.NewFromFloat(0.01)
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	cryptoTradeID, err := svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Fee:          &cryptoFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
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

	// Buy 5 IWDA at 100 EUR with 3 EUR fee = value 497 EUR
	stockAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	stockAssetID, err := svc.InsertAsset(ctx3, userID, stockAssetReq)
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	stockFee := decimal.NewFromInt(3)
	ctx4, cancel4 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel4()

	stockTradeID, err := svc.InsertInvestmentTrade(ctx4, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &stockFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(stockTradeID, int64(0))

	// Verify stock asset: quantity = 5, value = 500 - 3 = 497
	var stockAsset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", stockAssetID).First(&stockAsset).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(5).Equal(stockAsset.Quantity),
		"stock quantity should be 5, got %s", stockAsset.Quantity.String())
	s.Assert().True(decimal.NewFromInt(497).Equal(stockAsset.ValueAtBuy),
		"stock value at buy should be 497, got %s", stockAsset.ValueAtBuy.String())
	s.Assert().True(decimal.NewFromFloat(99.4).Equal(stockAsset.AverageBuyPrice),
		"stock avg buy price should be 99.4, got %s", stockAsset.AverageBuyPrice.String())

	// Verify total cash outflows = 49500 (crypto) + 497 (stock) = 49997
	var finalBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&finalBalance).Error
	s.Require().NoError(err)

	expectedTotalOutflows := decimal.NewFromFloat(49997)
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	stockAssetID, err := svc.InsertAsset(ctx, userID, stockAssetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 10 shares at 100 EUR with 5 EUR fee = value 995
	buyFee := decimal.NewFromInt(5)
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &buyFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 5 shares at 120 EUR with 3 EUR fee
	// Proceeds = (5 * 120) - 3 = 597
	// Cost basis = 5 * 99.5 = 497.5
	// Realized P&L = 597 - 497.5 = 99.5
	sellFee := decimal.NewFromInt(3)
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	sellTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(120),
		Fee:          &sellFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var stockSellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", sellTradeID).First(&stockSellTrade).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(597).Equal(stockSellTrade.RealizedValue),
		"stock realized value should be 597 (proceeds after fee)")

	// Verify balance: buy wrote 995 outflows, sell wrote 99.5 inflows
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balance).Error
	s.Require().NoError(err)

	expectedOutflows := decimal.NewFromFloat(995)
	s.Assert().True(expectedOutflows.Equal(balance.CashOutflows),
		"cash outflows should be %s, got %s", expectedOutflows.String(), balance.CashOutflows.String())

	expectedInflows := decimal.NewFromFloat(99.5)
	s.Assert().True(expectedInflows.Equal(balance.CashInflows),
		"cash inflows (realized P&L) should be %s, got %s", expectedInflows.String(), balance.CashInflows.String())

	// Crypto with fee (fee in tokens)
	cryptoAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-EUR",
		Currency:       "EUR",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx4, cancel4 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel4()

	cryptoAssetID, err := svc.InsertAsset(ctx4, userID, cryptoAssetReq)
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 1 BTC with 0.01 BTC fee -> effective 0.99 BTC
	cryptoBuyFee := decimal.NewFromFloat(0.01)
	ctx5, cancel5 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel5()

	_, err = svc.InsertInvestmentTrade(ctx5, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Fee:          &cryptoBuyFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx5.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 0.5 BTC at 90k with 0.005 BTC fee
	// Effective sell qty = 0.5 - 0.005 = 0.495
	// Realized value = 0.495 * 90k = 44,550
	cryptoSellFee := decimal.NewFromFloat(0.005)
	ctx6, cancel6 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel6()

	cryptoSellTradeID, err := svc.InsertInvestmentTrade(ctx6, userID, &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromFloat(0.5),
		PricePerUnit: decimal.NewFromInt(90000),
		Fee:          &cryptoSellFee,
		Currency:     "EUR",
	})
	if err != nil {
		if ctx6.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	s.Assert().True(decimal.NewFromFloat(0.495).Equal(cryptoAssetAfterSell.Quantity),
		"remaining crypto quantity should be 0.495, got %s", cryptoAssetAfterSell.Quantity.String())
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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 2 BTC at 50k EUR
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1 BTC at 90k EUR (40k profit)
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	sellTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Verify: balance should have 100k outflows (buy) and 40k inflows (realized P&L)
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceBeforeDelete.CashOutflows),
		"cash outflows should be 100k, got %s", balanceBeforeDelete.CashOutflows.String())
	s.Assert().True(decimal.NewFromInt(40000).Equal(balanceBeforeDelete.CashInflows),
		"cash inflows should be 40k, got %s", balanceBeforeDelete.CashInflows.String())

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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 1: 1 BTC at 50k EUR
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 2: 1 BTC at 60k EUR
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	secondBuyTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 1: 1 BTC at 50k
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	firstBuyTradeID, err := svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 2: 1 BTC at 60k
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	_, err = svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1.5 BTC
	ctx4, cancel4 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel4()

	_, err = svc.InsertInvestmentTrade(ctx4, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromFloat(1.5),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

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

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := svc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Buy 2 BTC at 50k EUR each = 100k outflows
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1 BTC at 90k EUR = 40k realized P&L inflows
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	_, err = svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Verify balance before delete: 100k outflows, 40k inflows, end = 140k
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(100000).Equal(balanceBeforeDelete.CashOutflows),
		"cash outflows should be 100k before delete")
	s.Assert().True(decimal.NewFromInt(40000).Equal(balanceBeforeDelete.CashInflows),
		"cash inflows should be 40k before delete")
	s.Assert().True(decimal.NewFromInt(140000).Equal(balanceBeforeDelete.EndBalance),
		"end balance should be 140k before delete")

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
