package services_test

import (
	"context"
	"errors"
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

	// Create account 6 months ago with 50k balance
	sixMonthsAgo := time.Now().AddDate(0, -6, 0).UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(50000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      sixMonthsAgo,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create BTC asset
	qty := decimal.NewFromInt(0)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Currency:       "USD",
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

	// Execute buy trade: 1 BTC at 50k, 6 months ago
	buyQty := decimal.NewFromInt(1)
	buyPrice := decimal.NewFromInt(50000)
	tradeReq := &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      sixMonthsAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     buyQty,
		PricePerUnit: buyPrice,
		Currency:     "USD",
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

	// Verify asset was updated
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", assetID).
		First(&asset).Error
	s.Require().NoError(err)

	s.Assert().True(buyQty.Equal(asset.Quantity))
	s.Assert().True(buyPrice.Equal(asset.AverageBuyPrice))
	s.Assert().True(buyPrice.Equal(asset.ValueAtBuy))

	// Generate checkpoints using same logic as service
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var checkpoints []time.Time

	// Trade date adjusted to weekday
	checkpoints = append(checkpoints, utils.AdjustToWeekday(sixMonthsAgo))

	// First day of each month between trade date and today
	current := sixMonthsAgo.AddDate(0, 1, 0)
	current = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)

	for current.Before(today) {
		adjusted := utils.AdjustToWeekday(current)
		if !adjusted.Equal(checkpoints[len(checkpoints)-1]) {
			checkpoints = append(checkpoints, adjusted)
		}
		current = current.AddDate(0, 1, 0)
	}

	adjustedToday := utils.AdjustToWeekday(today)
	if !adjustedToday.Equal(checkpoints[len(checkpoints)-1]) {
		checkpoints = append(checkpoints, adjustedToday)
	}

	// Verify balance records exist for each checkpoint
	for _, checkpoint := range checkpoints {
		var balance models.Balance
		err = s.TC.DB.WithContext(s.Ctx).
			Where("account_id = ? AND as_of = ?", accID, checkpoint).
			First(&balance).Error
		s.Require().NoError(err, "balance record should exist for checkpoint %s", checkpoint.Format("2006-01-02"))
	}

	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)

	expectedBalanceUSD := asset.CurrentValue
	exchangeRate := svc.GetExchangeRate(s.Ctx, "USD", "EUR")
	expectedBalanceAccCurrency := expectedBalanceUSD.Mul(exchangeRate)

	expectedTotalBalance := initialBalance.Add(expectedBalanceAccCurrency.Sub(decimal.NewFromInt(50000).Mul(exchangeRate)))

	s.Assert().True(expectedTotalBalance.Sub(latestBalance.EndBalance).Abs().LessThan(decimal.NewFromFloat(1.0)),
		"account balance should equal initial balance plus P&L: expected ~%s EUR, got %s EUR",
		expectedTotalBalance.StringFixed(2), latestBalance.EndBalance.StringFixed(2))
}

// Tests that multiple buy trades correctly update the weighted average buy price and track unrealized P&L
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_MultipleBuysUpdateAverage() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	sixMonthsAgo := time.Now().AddDate(0, -6, 0).UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      sixMonthsAgo,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromInt(0)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       qty,
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

	// First buy: 1 BTC at 50k
	trade1Date := sixMonthsAgo
	trade1Qty := decimal.NewFromInt(1)
	trade1Price := decimal.NewFromInt(50000)

	ctx1, cancel1 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel1()

	trade1ID, err := svc.InsertInvestmentTrade(ctx1, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      trade1Date,
		TradeType:    models.InvestmentBuy,
		Quantity:     trade1Qty,
		PricePerUnit: trade1Price,
		Currency:     "USD",
	})
	if err != nil {
		if ctx1.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var trade1 models.InvestmentTrade
	s.TC.DB.WithContext(s.Ctx).Where("id = ?", trade1ID).First(&trade1)

	// Second buy: 0.5 BTC at 60k (3 months ago)
	trade2Date := time.Now().AddDate(0, -3, 0).UTC().Truncate(24 * time.Hour)
	trade2Qty := decimal.NewFromFloat(0.5)
	trade2Price := decimal.NewFromInt(60000)

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	trade2ID, err := svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      trade2Date,
		TradeType:    models.InvestmentBuy,
		Quantity:     trade2Qty,
		PricePerUnit: trade2Price,
		Currency:     "USD",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var trade2 models.InvestmentTrade
	s.TC.DB.WithContext(s.Ctx).Where("id = ?", trade2ID).First(&trade2)

	// Third buy: 0.5 BTC at 55k (1 month ago)
	trade3Date := time.Now().AddDate(0, -1, 0).UTC().Truncate(24 * time.Hour)
	trade3Qty := decimal.NewFromFloat(0.5)
	trade3Price := decimal.NewFromInt(55000)

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	trade3ID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      trade3Date,
		TradeType:    models.InvestmentBuy,
		Quantity:     trade3Qty,
		PricePerUnit: trade3Price,
		Currency:     "USD",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var trade3 models.InvestmentTrade
	s.TC.DB.WithContext(s.Ctx).Where("id = ?", trade3ID).First(&trade3)

	// Verify final asset state
	var asset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", assetID).
		First(&asset).Error
	s.Require().NoError(err)

	expectedQty := decimal.NewFromInt(2)
	s.Assert().True(expectedQty.Equal(asset.Quantity),
		"total quantity should be %s, got %s", expectedQty.String(), asset.Quantity.String())

	expectedAvgPrice := decimal.NewFromInt(53750)
	s.Assert().True(expectedAvgPrice.Equal(asset.AverageBuyPrice),
		"average buy price should be %s, got %s", expectedAvgPrice.String(), asset.AverageBuyPrice.String())

	expectedValueAtBuy := decimal.NewFromInt(107500)
	s.Assert().True(expectedValueAtBuy.Equal(asset.ValueAtBuy),
		"value at buy should be %s, got %s", expectedValueAtBuy.String(), asset.ValueAtBuy.String())

	// Verify P&L calculation
	expectedCurrentValue := asset.Quantity.Mul(*asset.CurrentPrice)
	s.Assert().True(expectedCurrentValue.Sub(asset.CurrentValue).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"current value should be close to %s, got %s (diff: %s)",
		expectedCurrentValue.String(), asset.CurrentValue.String(),
		expectedCurrentValue.Sub(asset.CurrentValue).Abs().String())

	expectedPnL := asset.CurrentValue.Sub(asset.ValueAtBuy)
	s.Assert().True(expectedPnL.Equal(asset.ProfitLoss),
		"profit/loss should be %s, got %s", expectedPnL.String(), asset.ProfitLoss.String())

	var allBalances []models.Balance
	s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of ASC").
		Find(&allBalances)

	// Verify account balance
	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)

	var firstBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of ASC").
		First(&firstBalance).Error
	s.Require().NoError(err)

	totalSpentUSD := decimal.NewFromInt(107500)
	exchangeRate := svc.GetExchangeRate(s.Ctx, "USD", "EUR")
	totalSpentAccCurrency := totalSpentUSD.Mul(exchangeRate)

	// Convert current asset value from USD to acc currency
	assetValueAccCurrency := asset.CurrentValue.Mul(exchangeRate)

	expectedBalance := initialBalance.Sub(totalSpentAccCurrency).Add(assetValueAccCurrency)

	s.Assert().True(expectedBalance.Sub(latestBalance.EndBalance).Abs().LessThan(decimal.NewFromFloat(1.0)),
		"account balance should be initial - spent + asset value: expected ~%s EUR, got %s EUR",
		expectedBalance.StringFixed(2), latestBalance.EndBalance.StringFixed(2))
}

// Tests that selling an investment records realized gains/losses as cash inflows/outflows in the balance
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_SellRecordsRealizedPnL() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	sixMonthsAgo := time.Now().AddDate(0, -6, 0).UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(200000)
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      sixMonthsAgo,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	qty := decimal.NewFromInt(0)
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       qty,
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

	// Execute 3 buy trades
	trades := []struct {
		date  time.Time
		qty   decimal.Decimal
		price decimal.Decimal
	}{
		{sixMonthsAgo, decimal.NewFromInt(1), decimal.NewFromInt(50000)},
		{time.Now().AddDate(0, -3, 0).UTC().Truncate(24 * time.Hour), decimal.NewFromFloat(0.5), decimal.NewFromInt(60000)},
		{time.Now().AddDate(0, -1, 0).UTC().Truncate(24 * time.Hour), decimal.NewFromFloat(0.5), decimal.NewFromInt(55000)},
	}

	for _, trade := range trades {
		ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
		defer cancel()

		_, err = svc.InsertInvestmentTrade(ctx, userID, &models.InvestmentTradeReq{
			AssetID:      assetID,
			TxnDate:      trade.date,
			TradeType:    models.InvestmentBuy,
			Quantity:     trade.qty,
			PricePerUnit: trade.price,
			Currency:     "USD",
		})
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				s.T().Skip("Skipping test: price fetch timed out")
			}
			s.Require().NoError(err)
		}
	}

	// Verify we have 2 BTC with avg price of 53,750
	var assetBeforeSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeSell).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(2).Equal(assetBeforeSell.Quantity))
	s.Assert().True(decimal.NewFromInt(53750).Equal(assetBeforeSell.AverageBuyPrice))

	// Now sell 1 BTC at higher market price
	today := time.Now().UTC().Truncate(24 * time.Hour)
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
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(sellTradeID, int64(0))

	// Verify asset updated
	var assetAfterSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterSell).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(1).Equal(assetAfterSell.Quantity))
	s.Assert().True(decimal.NewFromInt(53750).Equal(assetAfterSell.AverageBuyPrice))

	// Calculate expected realized P&L
	proceeds := sellQty.Mul(sellPrice)
	costBasis := assetBeforeSell.AverageBuyPrice.Mul(sellQty)
	expectedRealizedPnLUSD := proceeds.Sub(costBasis)

	// Convert to account currency
	exchangeRate := svc.GetExchangeRate(s.Ctx, "USD", "EUR")
	expectedRealizedPnLAccCurrency := expectedRealizedPnLUSD.Mul(exchangeRate)

	var todayBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&todayBalance).Error
	s.Require().NoError(err)

	s.Assert().True(expectedRealizedPnLAccCurrency.Sub(todayBalance.CashInflows).Abs().LessThan(decimal.NewFromFloat(1.0)),
		"realized gains should be recorded as cash inflows: expected ~%s (acc currency), got %s",
		expectedRealizedPnLAccCurrency.StringFixed(2), todayBalance.CashInflows.StringFixed(2))

	// Verify final account balance reflects both unrealized and realized gains
	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)

	// Get balance right before the sell
	var balanceBeforeSell models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of < ?", accID, today).
		Order("as_of DESC").
		First(&balanceBeforeSell).Error
	s.Require().NoError(err)

	// Balance should have increased by at least the realized gain (with some tolerance for price fluctuations)
	s.Assert().True(latestBalance.EndBalance.GreaterThan(balanceBeforeSell.EndBalance.Add(expectedRealizedPnLAccCurrency).Sub(decimal.NewFromInt(1000))),
		"balance should increase by at least realized gains")
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
	cryptoTradeReq := &models.InvestmentTradeReq{
		AssetID:      cryptoAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Fee:          &cryptoFee,
		Currency:     "USD",
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	cryptoTradeID, err := svc.InsertInvestmentTrade(ctx2, userID, cryptoTradeReq)
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(cryptoTradeID, int64(0))

	// Verify crypto asset: quantity = 1 - 0.01 = 0.99 BTC
	var cryptoAsset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", cryptoAssetID).
		First(&cryptoAsset).Error
	s.Require().NoError(err)

	expectedCryptoQty := decimal.NewFromFloat(0.99)
	s.Assert().True(expectedCryptoQty.Equal(cryptoAsset.Quantity),
		"crypto quantity should be 0.99 (1 - 0.01 fee), got %s", cryptoAsset.Quantity.String())

	// Value at buy = 0.99 * 50,000 = 49,500
	expectedCryptoValue := decimal.NewFromFloat(49500)
	s.Assert().True(expectedCryptoValue.Equal(cryptoAsset.ValueAtBuy),
		"crypto value at buy should be 49,500, got %s", cryptoAsset.ValueAtBuy.String())

	// Buy 5 IWDA at 100 EUR with 3 EUR fee = value 497 EUR
	stockAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentStock,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
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
	stockTradeReq := &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &stockFee,
		Currency:     "EUR",
	}

	ctx4, cancel4 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel4()

	stockTradeID, err := svc.InsertInvestmentTrade(ctx4, userID, stockTradeReq)
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}
	s.Assert().Greater(stockTradeID, int64(0))

	// Verify stock asset: quantity = 5 (not affected by fee)
	var stockAsset models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", stockAssetID).
		First(&stockAsset).Error
	s.Require().NoError(err)

	expectedStockQty := decimal.NewFromInt(5)
	s.Assert().True(expectedStockQty.Equal(stockAsset.Quantity),
		"stock quantity should be 5, got %s", stockAsset.Quantity.String())

	expectedStockValue := decimal.NewFromInt(497)
	s.Assert().True(expectedStockValue.Equal(stockAsset.ValueAtBuy),
		"stock value at buy should be 497, got %s", stockAsset.ValueAtBuy.String())

	expectedStockAvgPrice := decimal.NewFromFloat(99.4)
	s.Assert().True(expectedStockAvgPrice.Equal(stockAsset.AverageBuyPrice),
		"stock average buy price should be 99.4, got %s", stockAsset.AverageBuyPrice.String())
}

// Tests that fees are correctly deducted from realized P&L when selling investments (both crypto and stocks)
func (s *InvestmentServiceTestSuite) TestInsertInvestmentTrade_SellWithFees() {
	svc := s.TC.App.InvestmentService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	for today.Weekday() != time.Wednesday {
		today = today.AddDate(0, 0, 1)
	}
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

	// Buy 10 shares at 100 with 5 fee
	buyFee := decimal.NewFromInt(5)
	buyTradeReq := &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(100),
		Fee:          &buyFee,
		Currency:     "EUR",
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, buyTradeReq)
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 5 shares at 120 with 3 fee
	// Proceeds = (5 * 120) - 3 = 597
	// Cost basis = 5 * 99.5 = 497.5
	// Realized P&L = 597 - 497.5 = 99.5
	sellFee := decimal.NewFromInt(3)
	sellTradeReq := &models.InvestmentTradeReq{
		AssetID:      stockAssetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(5),
		PricePerUnit: decimal.NewFromInt(120),
		Fee:          &sellFee,
		Currency:     "EUR",
	}

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	sellTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, sellTradeReq)
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var stockSellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", sellTradeID).
		First(&stockSellTrade).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(597).Equal(stockSellTrade.RealizedValue),
		"stock realized value should be 597 (proceeds after fee)")

	// Crypto with fee (fee in tokens, doesn't affect cash proceeds)
	cryptoAssetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
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

	// Buy 1 BTC with 0.01 BTC fee -> effective 0.99 BTC at 50k
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
		Currency:     "USD",
	})
	if err != nil {
		if ctx5.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1 BTC at 90k with 0.005 BTC fee
	// Proceeds = 0.995 * 90k = 89,550
	// Cost basis = 0.995 * 50,000 = 49,750
	// Realized P&L = 89,550 - 49,750 = 39,800
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
		Currency:     "USD",
	})
	if err != nil {
		if ctx6.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	var cryptoSellTrade models.InvestmentTrade
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", cryptoSellTradeID).
		First(&cryptoSellTrade).Error
	s.Require().NoError(err)

	// Effective quantity = 1 - 0.005 = 0.995
	expectedCryptoRealizedValue := decimal.NewFromFloat(0.495).Mul(decimal.NewFromInt(90000))

	s.Assert().True(expectedCryptoRealizedValue.Equal(cryptoSellTrade.RealizedValue),
		"crypto realized value should account for token fee")

	var cryptoAssetAfterSell models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", cryptoAssetID).
		First(&cryptoAssetAfterSell).Error
	s.Require().NoError(err)

	expectedRemainingQty := decimal.NewFromFloat(0.495)
	s.Assert().True(expectedRemainingQty.Equal(cryptoAssetAfterSell.Quantity),
		"remaining crypto quantity should be 0.495")
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
		Ticker:         "BTC-USD",
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

	// Buy 2 BTC at 50k
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1 BTC at 90k (realize ~40k profit)
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	sellTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Verify asset has 1 BTC remaining
	var assetBeforeDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(1).Equal(assetBeforeDelete.Quantity))

	// Verify realized gain recorded
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

	realizedGain := balanceBeforeDelete.CashInflows
	s.Assert().True(realizedGain.GreaterThan(decimal.Zero), "should have realized gain")

	// Delete the sell trade
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, sellTradeID)
	s.Require().NoError(err)

	// Verify asset recalculated: should have 2 BTC again
	var assetAfterDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(2).Equal(assetAfterDelete.Quantity),
		"quantity should be restored to 2 BTC, got %s", assetAfterDelete.Quantity.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.AverageBuyPrice),
		"average buy price should be 50k, got %s", assetAfterDelete.AverageBuyPrice.String())

	// Verify realized gain reversed
	var balanceAfterDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterDelete).Error
	s.Require().NoError(err)

	// Cash inflows should be zero or negative (reversed)
	s.Assert().True(balanceAfterDelete.CashInflows.LessThan(realizedGain),
		"realized gain should be reversed: before %s, after %s",
		realizedGain.String(), balanceAfterDelete.CashInflows.String())

	// Verify sell trade deleted
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
		Ticker:         "BTC-USD",
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

	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "USD",
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

	secondBuyTradeID, err := svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Verify asset has 2 BTC with average price 55k
	var assetBeforeDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(2).Equal(assetBeforeDelete.Quantity))
	s.Assert().True(decimal.NewFromInt(55000).Equal(assetBeforeDelete.AverageBuyPrice),
		"average should be 55k: (1*50k + 1*60k)/2")

	// Delete the second buy trade
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, secondBuyTradeID)
	s.Require().NoError(err)

	// Verify asset recalculated: should have 1 BTC at 50k
	var assetAfterDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterDelete).Error
	s.Require().NoError(err)

	s.Assert().True(decimal.NewFromInt(1).Equal(assetAfterDelete.Quantity),
		"quantity should be 1 BTC after deleting second buy, got %s", assetAfterDelete.Quantity.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.AverageBuyPrice),
		"average price should be 50k (from first buy), got %s", assetAfterDelete.AverageBuyPrice.String())
	s.Assert().True(decimal.NewFromInt(50000).Equal(assetAfterDelete.ValueAtBuy),
		"value at buy should be 50k, got %s", assetAfterDelete.ValueAtBuy.String())

	// Verify trade deleted
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
		Ticker:         "BTC-USD",
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
		Currency:     "USD",
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
		Currency:     "USD",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1.5 BTC at 90k (requires both buys to have enough quantity)
	ctx4, cancel4 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel4()

	_, err = svc.InsertInvestmentTrade(ctx4, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromFloat(1.5),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Verify current state: 0.5 BTC remaining
	var assetBeforeDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetBeforeDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromFloat(0.5).Equal(assetBeforeDelete.Quantity))

	// Try to delete the first buy trade
	// This should fail because it would leave only 1 BTC from the second buy,
	// but 1.5 BTC was already sold, which would be impossible
	err = svc.DeleteInvestmentTrade(s.Ctx, userID, firstBuyTradeID)

	s.Require().Error(err, "should not allow deleting buy that would make sell invalid")

	// Verify asset unchanged
	var assetAfterFailedDelete models.InvestmentAsset
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", assetID).First(&assetAfterFailedDelete).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromFloat(0.5).Equal(assetAfterFailedDelete.Quantity),
		"quantity should remain 0.5 BTC")

	// Verify trade still exists
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
		Ticker:         "BTC-USD",
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

	// Create multiple trades
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	// Buy 2 BTC
	_, err = svc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Sell 1 BTC
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel3()

	_, err = svc.InsertInvestmentTrade(ctx3, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentSell,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "USD",
	})
	if err != nil {
		if ctx3.Err() == context.DeadlineExceeded {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Count trades before delete
	var tradeCountBefore int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.InvestmentTrade{}).
		Where("asset_id = ?", assetID).
		Count(&tradeCountBefore).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), tradeCountBefore, "should have 2 trades")

	// Get balance before delete
	var balanceBeforeDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceBeforeDelete).Error
	s.Require().NoError(err)

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

	// Verify balance updated
	var balanceAfterDelete models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, today).
		First(&balanceAfterDelete).Error
	s.Require().NoError(err)

	// Realized P&L should be reversed
	s.Assert().True(balanceAfterDelete.CashInflows.LessThan(balanceBeforeDelete.CashInflows),
		"realized gains should be reversed")

	// Unrealized P&L should be cleared (or significantly reduced)
	s.Assert().True(balanceAfterDelete.NonCashInflows.LessThanOrEqual(balanceBeforeDelete.NonCashInflows),
		"unrealized gains should be cleared or reduced")

	// Final balance should be close to initial balance
	s.Assert().True(balanceAfterDelete.EndBalance.LessThanOrEqual(balanceBeforeDelete.EndBalance),
		"end balance should be reduced after deleting asset")
}
