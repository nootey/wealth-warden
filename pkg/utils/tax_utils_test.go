package utils_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var taxToday = time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)

func daysAgo(n int) time.Time { return taxToday.AddDate(0, 0, -n) }

func df(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func bracket(min int, taxPct float64) models.InvestmentTaxBracket {
	return models.InvestmentTaxBracket{MinDaysHeld: min, TaxablePercent: df(taxPct)}
}

func buyTrade(id int64, daysHeld int, qty, valueAtBuy, fee, profitLoss float64) models.InvestmentTrade {
	return models.InvestmentTrade{
		ID:         id,
		TradeType:  models.InvestmentBuy,
		TxnDate:    daysAgo(daysHeld),
		Quantity:   df(qty),
		ValueAtBuy: df(valueAtBuy),
		Fee:        df(fee),
		ProfitLoss: df(profitLoss),
	}
}

func sellTrade(id int64, daysHeld int, qty float64) models.InvestmentTrade {
	return models.InvestmentTrade{
		ID:        id,
		TradeType: models.InvestmentSell,
		TxnDate:   daysAgo(daysHeld),
		Quantity:  df(qty),
	}
}

// --- ApplyBracket ---

func TestApplyBracket_Empty(t *testing.T) {
	assert.Nil(t, utils.ApplyBracket(nil, 100))
}

func TestApplyBracket_NoMatch(t *testing.T) {
	brackets := []models.InvestmentTaxBracket{bracket(10, 100)}
	assert.Nil(t, utils.ApplyBracket(brackets, 5))
}

func TestApplyBracket_ExactMatch(t *testing.T) {
	brackets := []models.InvestmentTaxBracket{bracket(0, 100), bracket(1826, 0)}
	b := utils.ApplyBracket(brackets, 1826)
	assert.NotNil(t, b)
	assert.True(t, b.TaxablePercent.IsZero())
}

func TestApplyBracket_PicksHighestMatchingStep(t *testing.T) {
	brackets := []models.InvestmentTaxBracket{
		bracket(0, 100),
		bracket(365, 50),
		bracket(1826, 0),
	}

	tests := []struct {
		days    int
		wantPct float64
	}{
		{0, 100},
		{364, 100},
		{365, 50},
		{1825, 50},
		{1826, 0},
		{5000, 0},
	}

	for _, tc := range tests {
		b := utils.ApplyBracket(brackets, tc.days)
		assert.NotNil(t, b)
		assert.True(t, df(tc.wantPct).Equal(b.TaxablePercent), "days=%d want pct=%.0f got %s", tc.days, tc.wantPct, b.TaxablePercent)
	}
}

// --- BuildFifoLots ---

func TestBuildFifoLots_SingleBuy(t *testing.T) {
	trades := []models.InvestmentTrade{buyTrade(1, 100, 10, 100, 0, 0)}
	lots := utils.BuildFifoLots(trades, 0)
	assert.Len(t, lots, 1)
	assert.True(t, df(10).Equal(lots[0].Quantity))
}

func TestBuildFifoLots_BuySellAll(t *testing.T) {
	trades := []models.InvestmentTrade{
		buyTrade(1, 100, 10, 100, 0, 0),
		sellTrade(2, 50, 10),
	}
	lots := utils.BuildFifoLots(trades, 0)
	assert.Len(t, lots, 1)
	assert.True(t, lots[0].Quantity.IsZero())
}

func TestBuildFifoLots_PartialSell(t *testing.T) {
	trades := []models.InvestmentTrade{
		buyTrade(1, 100, 10, 100, 0, 0),
		sellTrade(2, 50, 4),
	}
	lots := utils.BuildFifoLots(trades, 0)
	assert.True(t, df(6).Equal(lots[0].Quantity))
	assert.True(t, df(60).Equal(lots[0].ValueAtBuy))
}

func TestBuildFifoLots_FifoOrdering(t *testing.T) {
	// buy 5 (lot1) + buy 5 (lot2), sell 7 → lot1 fully consumed, lot2 has 3 left
	trades := []models.InvestmentTrade{
		buyTrade(1, 200, 5, 50, 0, 0),
		buyTrade(2, 100, 5, 50, 0, 0),
		sellTrade(3, 50, 7),
	}
	lots := utils.BuildFifoLots(trades, 0)
	assert.True(t, lots[0].Quantity.IsZero(), "lot1 should be fully consumed")
	assert.True(t, df(3).Equal(lots[1].Quantity), "lot2 should have 3 remaining")
}

func TestBuildFifoLots_StopAtID(t *testing.T) {
	// buy(id=1), sell(id=2): stopAtID=2 should halt before the sell
	trades := []models.InvestmentTrade{
		buyTrade(1, 200, 10, 100, 0, 0),
		sellTrade(2, 100, 5),
	}
	lots := utils.BuildFifoLots(trades, 2)
	assert.True(t, df(10).Equal(lots[0].Quantity), "sell should not be applied when stopAtID=2")
}

// --- ComputeBuyTradeTaxInfo ---

var sloETFBrackets = []models.InvestmentTaxBracket{
	bracket(0, 27.5),
	bracket(1826, 0),
}

func TestComputeBuyTradeTaxInfo_NoBrackets(t *testing.T) {
	trade := buyTrade(1, 500, 10, 100, 0, 50)
	info := utils.ComputeBuyTradeTaxInfo(trade, nil, taxToday)
	assert.Equal(t, 500, info.DaysHeld)
	assert.Nil(t, info.TaxablePercent)
	assert.True(t, info.TaxableProfit.IsZero())
}

func TestComputeBuyTradeTaxInfo_PositivePnL(t *testing.T) {
	trade := buyTrade(1, 500, 10, 100, 0, 80)
	info := utils.ComputeBuyTradeTaxInfo(trade, sloETFBrackets, taxToday)

	assert.Equal(t, 500, info.DaysHeld)
	assert.NotNil(t, info.TaxablePercent)
	assert.True(t, df(27.5).Equal(*info.TaxablePercent))
	// taxable profit = 80 * 27.5 / 100 = 22
	assert.True(t, df(22).Equal(info.TaxableProfit))
}

func TestComputeBuyTradeTaxInfo_NegativePnLNotTaxed(t *testing.T) {
	trade := buyTrade(1, 500, 10, 200, 0, -50)
	info := utils.ComputeBuyTradeTaxInfo(trade, sloETFBrackets, taxToday)
	assert.True(t, info.TaxableProfit.IsZero())
}

func TestComputeBuyTradeTaxInfo_DaysUntilNextBracket(t *testing.T) {
	trade := buyTrade(1, 500, 10, 100, 0, 50)
	info := utils.ComputeBuyTradeTaxInfo(trade, sloETFBrackets, taxToday)
	assert.NotNil(t, info.DaysUntilNextBracket)
	assert.Equal(t, 1826-500, *info.DaysUntilNextBracket)
}

func TestComputeBuyTradeTaxInfo_AlreadyAtTaxFreeBracket(t *testing.T) {
	trade := buyTrade(1, 2000, 10, 100, 0, 50)
	info := utils.ComputeBuyTradeTaxInfo(trade, sloETFBrackets, taxToday)

	assert.NotNil(t, info.TaxablePercent)
	assert.True(t, info.TaxablePercent.IsZero())
	assert.True(t, info.TaxableProfit.IsZero())
	assert.NotNil(t, info.DaysUntilTaxFree)
	assert.Equal(t, 0, *info.DaysUntilTaxFree)
	assert.Nil(t, info.DaysUntilNextBracket)
}

func TestComputeBuyTradeTaxInfo_DaysUntilTaxFree(t *testing.T) {
	trade := buyTrade(1, 500, 10, 100, 0, 50)
	info := utils.ComputeBuyTradeTaxInfo(trade, sloETFBrackets, taxToday)
	assert.NotNil(t, info.DaysUntilTaxFree)
	assert.Equal(t, 1826-500, *info.DaysUntilTaxFree)
}

func TestComputeBuyTradeTaxInfo_NoTaxFreeBracket(t *testing.T) {
	brackets := []models.InvestmentTaxBracket{bracket(0, 27.5)}
	trade := buyTrade(1, 100, 10, 100, 0, 50)
	info := utils.ComputeBuyTradeTaxInfo(trade, brackets, taxToday)
	assert.Nil(t, info.DaysUntilTaxFree)
}

// --- ComputeSellTradeTaxInfo ---

func TestComputeSellTradeTaxInfo_SingleLot(t *testing.T) {
	// buy 200 days ago, sell today
	allTrades := []models.InvestmentTrade{
		buyTrade(1, 200, 10, 100, 0, 0),
	}
	sell := models.InvestmentTrade{
		ID:         2,
		TradeType:  models.InvestmentSell,
		TxnDate:    taxToday,
		Quantity:   df(10),
		ProfitLoss: df(50),
	}

	info := utils.ComputeSellTradeTaxInfo(sell, allTrades, sloETFBrackets)

	assert.Equal(t, 200, info.DaysHeld)
	assert.NotNil(t, info.TaxablePercent)
	assert.True(t, df(27.5).Equal(*info.TaxablePercent))
	// taxable profit = 50 * 27.5 / 100 = 13.75
	assert.True(t, df(13.75).Equal(info.TaxableProfit))
}

func TestComputeSellTradeTaxInfo_WeightedAvgDaysHeld(t *testing.T) {
	// lot1: 5 units, 100 days ago; lot2: 5 units, 300 days ago
	// sell 7 units → consumes all 5 from lot1 + 2 from lot2
	// weighted avg = (5*100 + 2*300) / 7 = 1100/7 ≈ 157 days
	allTrades := []models.InvestmentTrade{
		buyTrade(1, 100, 5, 50, 0, 0),
		buyTrade(2, 300, 5, 50, 0, 0),
	}
	sell := models.InvestmentTrade{
		ID:         3,
		TradeType:  models.InvestmentSell,
		TxnDate:    taxToday,
		Quantity:   df(7),
		ProfitLoss: df(0),
	}

	info := utils.ComputeSellTradeTaxInfo(sell, allTrades, sloETFBrackets)

	expected := (5*100 + 2*300) / 7
	assert.Equal(t, expected, info.DaysHeld)
}

func TestComputeSellTradeTaxInfo_NoBrackets(t *testing.T) {
	allTrades := []models.InvestmentTrade{buyTrade(1, 100, 10, 100, 0, 0)}
	sell := models.InvestmentTrade{
		ID:        2,
		TradeType: models.InvestmentSell,
		TxnDate:   taxToday,
		Quantity:  df(10),
	}

	info := utils.ComputeSellTradeTaxInfo(sell, allTrades, nil)
	assert.Nil(t, info.TaxablePercent)
	assert.True(t, info.TaxableProfit.IsZero())
}

// --- ComputeAssetTaxSummary ---

var summaryBrackets = []models.InvestmentTaxBracket{
	bracket(0, 100),
	bracket(1826, 0),
}

func assetWithPrice(pnl, price float64, invType models.InvestmentType) models.InvestmentAsset {
	p := df(price)
	return models.InvestmentAsset{
		ProfitLoss:     df(pnl),
		CurrentPrice:   &p,
		InvestmentType: invType,
	}
}

func TestComputeAssetTaxSummary_NoBrackets(t *testing.T) {
	asset := assetWithPrice(100, 15, models.InvestmentStock)
	info := utils.ComputeAssetTaxSummary(asset, nil, nil, models.InvestmentTaxSettings{}, taxToday)
	assert.True(t, df(100).Equal(info.AfterTaxPnL))
	assert.True(t, info.EstimatedTaxDue.IsZero())
}

func TestComputeAssetTaxSummary_NilCurrentPrice(t *testing.T) {
	asset := models.InvestmentAsset{ProfitLoss: df(100), InvestmentType: models.InvestmentStock}
	info := utils.ComputeAssetTaxSummary(asset, nil, summaryBrackets, models.InvestmentTaxSettings{}, taxToday)
	assert.True(t, df(100).Equal(info.AfterTaxPnL))
	assert.True(t, info.EstimatedTaxDue.IsZero())
}

func TestComputeAssetTaxSummary_WithoutLossOffsetting_GainTaxed(t *testing.T) {
	// 10 units bought 500 days ago at price 10 (valueAtBuy=100)
	// current price = 20 → lot pnl = 200-100 = 100, taxed at 100%
	trades := []models.InvestmentTrade{buyTrade(1, 500, 10, 100, 0, 0)}
	asset := assetWithPrice(100, 20, models.InvestmentStock)

	info := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: false}, taxToday)

	assert.True(t, df(100).Equal(info.EstimatedTaxDue))
	assert.True(t, df(0).Equal(info.AfterTaxPnL))
}

func TestComputeAssetTaxSummary_WithoutLossOffsetting_LossNotTaxed(t *testing.T) {
	// lot at a loss is skipped
	trades := []models.InvestmentTrade{buyTrade(1, 500, 10, 200, 0, 0)}
	asset := assetWithPrice(-100, 10, models.InvestmentStock)

	info := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: false}, taxToday)

	assert.True(t, info.EstimatedTaxDue.IsZero())
	assert.True(t, df(-100).Equal(info.AfterTaxPnL))
}

func TestComputeAssetTaxSummary_WithoutLossOffsetting_MixedLots(t *testing.T) {
	// lot1: 500 days, 10 units @10 → current 20 → pnl=+100, tax=100 (100% bracket)
	// lot2: 2000 days, 10 units @10 → current 20 → pnl=+100, tax=0 (0% bracket)
	trades := []models.InvestmentTrade{
		buyTrade(1, 500, 10, 100, 0, 0),
		buyTrade(2, 2000, 10, 100, 0, 0),
	}
	asset := assetWithPrice(200, 20, models.InvestmentStock)

	info := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: false}, taxToday)

	assert.True(t, df(100).Equal(info.EstimatedTaxDue))
	assert.True(t, df(100).Equal(info.AfterTaxPnL))
}

func TestComputeAssetTaxSummary_LossOffsetting_NetGainZero(t *testing.T) {
	// lot1: 10 units @20 → current 30 → pnl=+100
	// lot2: 10 units @40 → current 30 → pnl=-100
	// net pnl = 0 → no tax
	trades := []models.InvestmentTrade{
		buyTrade(1, 100, 10, 200, 0, 0),
		buyTrade(2, 200, 10, 400, 0, 0),
	}
	asset := assetWithPrice(0, 30, models.InvestmentStock)

	info := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: true}, taxToday)

	assert.True(t, info.EstimatedTaxDue.IsZero())
	assert.True(t, df(0).Equal(info.AfterTaxPnL))
}

func TestComputeAssetTaxSummary_LossOffsetting_ReducesTax(t *testing.T) {
	// Without offsetting: lot1 gain=100 taxed at 100%, lot2 loss ignored → tax=100
	// With offsetting: net gain=0 → tax=0
	trades := []models.InvestmentTrade{
		buyTrade(1, 100, 10, 200, 0, 0),
		buyTrade(2, 200, 10, 400, 0, 0),
	}
	asset := assetWithPrice(0, 30, models.InvestmentStock)

	without := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: false}, taxToday)
	with := utils.ComputeAssetTaxSummary(asset, trades, summaryBrackets, models.InvestmentTaxSettings{LossOffsettingEnabled: true}, taxToday)

	assert.True(t, without.EstimatedTaxDue.GreaterThan(with.EstimatedTaxDue))
}
