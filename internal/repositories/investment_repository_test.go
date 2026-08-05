package repositories

import (
	"testing"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func dec(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func day(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }

func basisBuy(d int, qty, valueAtBuy, fee float64) models.InvestmentTrade {
	return models.InvestmentTrade{
		TradeType:  models.InvestmentBuy,
		TxnDate:    day(d),
		Quantity:   dec(qty),
		ValueAtBuy: dec(valueAtBuy),
		Fee:        dec(fee),
	}
}

func basisSell(d int, qty, valueAtBuy, fee float64) models.InvestmentTrade {
	return models.InvestmentTrade{
		TradeType:  models.InvestmentSell,
		TxnDate:    day(d),
		Quantity:   dec(qty),
		ValueAtBuy: dec(valueAtBuy),
		Fee:        dec(fee),
	}
}

func assertTotals(t *testing.T, qty, valueAtBuy, fees decimal.Decimal, wantQty, wantValueAtBuy, wantFees float64) {
	t.Helper()
	assert.Equal(t, dec(wantQty).String(), qty.String(), "quantity")
	assert.Equal(t, dec(wantValueAtBuy).String(), valueAtBuy.String(), "value at buy")
	assert.Equal(t, dec(wantFees).String(), fees.String(), "fees")
}

func TestWalkTradeTotals_BuysAccumulate(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisBuy(2, 10, 1200, 5),
	}

	qty, valueAtBuy, fees := walkTradeTotals(trades, nil)

	assertTotals(t, qty, valueAtBuy, fees, 20, 2200, 10)
}

func TestWalkTradeTotals_SellReducesBasisAndFeesProportionally(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisBuy(2, 10, 1200, 5),
		// Sells 25% of the position, so basis and fees both keep 75%
		basisSell(3, 5, 550, 2),
	}

	qty, valueAtBuy, fees := walkTradeTotals(trades, nil)

	assertTotals(t, qty, valueAtBuy, fees, 15, 1650, 7.5)
}

func TestWalkTradeTotals_SellFeeDoesNotReduceBasis(t *testing.T) {
	withFee := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisSell(2, 5, 500, 25),
	}
	withoutFee := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisSell(2, 5, 500, 0),
	}

	_, valueAtBuyA, feesA := walkTradeTotals(withFee, nil)
	_, valueAtBuyB, feesB := walkTradeTotals(withoutFee, nil)

	assert.Equal(t, valueAtBuyB.String(), valueAtBuyA.String())
	assert.Equal(t, feesB.String(), feesA.String())
}

func TestWalkTradeTotals_FullExitZeroesOut(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisSell(2, 10, 1000, 5),
	}

	qty, valueAtBuy, fees := walkTradeTotals(trades, nil)

	assertTotals(t, qty, valueAtBuy, fees, 0, 0, 0)
}

func TestWalkTradeTotals_AsOfExcludesLaterTrades(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisBuy(5, 10, 1200, 5),
		basisSell(9, 5, 550, 2),
	}

	asOf := day(5)
	qty, valueAtBuy, fees := walkTradeTotals(trades, &asOf)

	assertTotals(t, qty, valueAtBuy, fees, 20, 2200, 10)
}

func TestWalkTradeTotals_AsOfBeforeFirstTrade(t *testing.T) {
	trades := []models.InvestmentTrade{basisBuy(5, 10, 1000, 5)}

	asOf := day(1)
	qty, valueAtBuy, fees := walkTradeTotals(trades, &asOf)

	assertTotals(t, qty, valueAtBuy, fees, 0, 0, 0)
}

func TestCostBasis_FeesCountExceptForCrypto(t *testing.T) {
	assert.Equal(t, dec(1005).String(), CostBasis(dec(1000), dec(5), models.InvestmentStock).String())
	assert.Equal(t, dec(1005).String(), CostBasis(dec(1000), dec(5), models.InvestmentETF).String())
	assert.Equal(t, dec(1000).String(), CostBasis(dec(1000), dec(5), models.InvestmentCrypto).String())
}
