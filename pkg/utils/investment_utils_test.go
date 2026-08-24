package utils

import (
	"errors"
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

	qty, valueAtBuy, fees := WalkTradeTotals(trades, nil)

	assertTotals(t, qty, valueAtBuy, fees, 20, 2200, 10)
}

func TestWalkTradeTotals_SellReducesBasisAndFeesProportionally(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisBuy(2, 10, 1200, 5),
		// Sells 25% of the position, so basis and fees both keep 75%
		basisSell(3, 5, 550, 2),
	}

	qty, valueAtBuy, fees := WalkTradeTotals(trades, nil)

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

	_, valueAtBuyA, feesA := WalkTradeTotals(withFee, nil)
	_, valueAtBuyB, feesB := WalkTradeTotals(withoutFee, nil)

	assert.Equal(t, valueAtBuyB.String(), valueAtBuyA.String())
	assert.Equal(t, feesB.String(), feesA.String())
}

func TestWalkTradeTotals_FullExitZeroesOut(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisSell(2, 10, 1000, 5),
	}

	qty, valueAtBuy, fees := WalkTradeTotals(trades, nil)

	assertTotals(t, qty, valueAtBuy, fees, 0, 0, 0)
}

func TestWalkTradeTotals_AsOfExcludesLaterTrades(t *testing.T) {
	trades := []models.InvestmentTrade{
		basisBuy(1, 10, 1000, 5),
		basisBuy(5, 10, 1200, 5),
		basisSell(9, 5, 550, 2),
	}

	asOf := day(5)
	qty, valueAtBuy, fees := WalkTradeTotals(trades, &asOf)

	assertTotals(t, qty, valueAtBuy, fees, 20, 2200, 10)
}

func TestWalkTradeTotals_AsOfBeforeFirstTrade(t *testing.T) {
	trades := []models.InvestmentTrade{basisBuy(5, 10, 1000, 5)}

	asOf := day(1)
	qty, valueAtBuy, fees := WalkTradeTotals(trades, &asOf)

	assertTotals(t, qty, valueAtBuy, fees, 0, 0, 0)
}

func TestCostBasis_FeesCountExceptForCrypto(t *testing.T) {
	assert.Equal(t, dec(1005).String(), CostBasis(dec(1000), dec(5), models.InvestmentStock).String())
	assert.Equal(t, dec(1005).String(), CostBasis(dec(1000), dec(5), models.InvestmentETF).String())
	assert.Equal(t, dec(1000).String(), CostBasis(dec(1000), dec(5), models.InvestmentCrypto).String())
}

func xirrDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func flow(y int, m time.Month, d int, amount string) CashFlow {
	return CashFlow{Date: xirrDay(y, m, d), Amount: decimal.RequireFromString(amount)}
}

func assertRate(t *testing.T, got decimal.Decimal, want string, tolerance string) {
	t.Helper()
	diff := got.Sub(decimal.RequireFromString(want)).Abs()
	if diff.GreaterThan(decimal.RequireFromString(tolerance)) {
		t.Fatalf("rate = %s, want %s (tolerance %s)", got, want, tolerance)
	}
}

func TestXIRR_MatchesExcelReference(t *testing.T) {
	flows := []CashFlow{
		flow(2008, time.January, 1, "-10000"),
		flow(2008, time.March, 1, "2750"),
		flow(2008, time.October, 30, "4250"),
		flow(2009, time.February, 15, "3250"),
		flow(2009, time.April, 1, "2750"),
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRate(t, rate, "0.373363", "0.000001")
}

func TestXIRR_SinglePeriodEqualsSimpleReturn(t *testing.T) {
	flows := []CashFlow{
		flow(2024, time.January, 1, "-1000"),
		flow(2024, time.December, 31, "1100"),
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRate(t, rate, "0.10", "0.000001")
}

func TestXIRR_LateDepositDoesNotInflateReturn(t *testing.T) {
	flows := []CashFlow{
		flow(2024, time.January, 1, "-1000"),
		flow(2026, time.January, 1, "-1000"),
		flow(2026, time.August, 24, "2200"),
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRate(t, rate, "0.058597", "0.000001")

	if rate.GreaterThanOrEqual(decimal.RequireFromString("0.10")) {
		t.Fatalf("rate %s should be below the naive 10%% cost-basis return", rate)
	}
}

func TestXIRR_HandlesLoss(t *testing.T) {
	flows := []CashFlow{
		flow(2024, time.January, 1, "-1000"),
		flow(2024, time.December, 31, "800"),
	}

	rate, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRate(t, rate, "-0.20", "0.000001")
}

func TestXIRR_UnsortedInputGivesSameResult(t *testing.T) {
	sorted := []CashFlow{
		flow(2024, time.January, 1, "-1000"),
		flow(2024, time.July, 1, "-500"),
		flow(2025, time.January, 1, "1700"),
	}
	shuffled := []CashFlow{sorted[2], sorted[0], sorted[1]}

	want, err := XIRR(sorted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := XIRR(shuffled)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("shuffled = %s, sorted = %s", got, want)
	}
}

func TestXIRR_ReportsRateAboveCeiling(t *testing.T) {
	flows := []CashFlow{
		flow(2025, time.May, 5, "-1"),
		flow(2026, time.August, 12, "-100"),
		flow(2026, time.August, 24, "928810"),
	}

	_, err := XIRR(flows)
	if !errors.Is(err, ErrXIRRAboveRange) {
		t.Fatalf("err = %v, want %v", err, ErrXIRRAboveRange)
	}
}

func TestXIRR_Rejects(t *testing.T) {
	cases := []struct {
		name  string
		flows []CashFlow
		want  error
	}{
		{
			name:  "single flow",
			flows: []CashFlow{flow(2024, time.January, 1, "-1000")},
			want:  ErrXIRRNotEnoughFlows,
		},
		{
			name: "no sign change",
			flows: []CashFlow{
				flow(2024, time.January, 1, "-1000"),
				flow(2024, time.December, 31, "-500"),
			},
			want: ErrXIRRNoSignChange,
		},
		{
			name: "span under thirty days",
			flows: []CashFlow{
				flow(2024, time.January, 1, "-1000"),
				flow(2024, time.January, 4, "1020"),
			},
			want: ErrXIRRSpanTooShort,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := XIRR(tc.flows)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
