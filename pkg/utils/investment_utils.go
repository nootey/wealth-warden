package utils

import (
	"errors"
	"math"
	"sort"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
)

type TradeExchangeRateKey struct {
	From, To string
	Day      string
}

func NewTradeExchangeRateKey(trade models.InvestmentTrade) TradeExchangeRateKey {
	return TradeExchangeRateKey{
		From: trade.Currency,
		To:   trade.Asset.Account.Currency,
		Day:  trade.TxnDate.UTC().Format("2006-01-02"),
	}
}

func MergeStakingIntoTrades(trades []models.InvestmentTrade, income []models.InvestmentIncome) []models.InvestmentTrade {
	if len(income) == 0 {
		return trades
	}

	merged := make([]models.InvestmentTrade, 0, len(trades)+len(income))
	merged = append(merged, trades...)
	for _, inc := range income {
		if inc.IncomeType != models.IncomeTypeStaking || inc.Quantity == nil {
			continue
		}
		merged = append(merged, models.InvestmentTrade{
			TradeType:  models.InvestmentBuy,
			TxnDate:    inc.TxnDate,
			Quantity:   *inc.Quantity,
			ValueAtBuy: inc.Amount,
			CreatedAt:  inc.CreatedAt,
		})
	}

	// txn_date is a date, so same-day events fall back to entry order
	sort.SliceStable(merged, func(i, j int) bool {
		if !merged[i].TxnDate.Equal(merged[j].TxnDate) {
			return merged[i].TxnDate.Before(merged[j].TxnDate)
		}
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})

	return merged
}

func WalkTradeTotals(trades []models.InvestmentTrade, asOf *time.Time) (quantity, valueAtBuy, fees decimal.Decimal) {
	for _, txn := range trades {
		if asOf != nil && txn.TxnDate.After(*asOf) {
			break
		}

		if txn.TradeType == models.InvestmentBuy {
			quantity = quantity.Add(txn.Quantity)
			valueAtBuy = valueAtBuy.Add(txn.ValueAtBuy)
			fees = fees.Add(txn.Fee)
			continue
		}

		// Sell: reduce proportionally
		quantity = quantity.Sub(txn.Quantity)
		if quantity.GreaterThan(decimal.Zero) {
			soldProportion := txn.Quantity.Div(quantity.Add(txn.Quantity))
			remaining := decimal.NewFromInt(1).Sub(soldProportion)
			valueAtBuy = valueAtBuy.Mul(remaining)
			fees = fees.Mul(remaining)
		} else {
			valueAtBuy = decimal.Zero
			fees = decimal.Zero
		}
	}

	return quantity, valueAtBuy, fees
}

func CostBasis(valueAtBuy, fees decimal.Decimal, investmentType models.InvestmentType) decimal.Decimal {
	if investmentType == models.InvestmentCrypto {
		return valueAtBuy
	}
	return valueAtBuy.Add(fees)
}

func IsExtremePriceDrop(oldPrice *decimal.Decimal, newPrice decimal.Decimal) bool {
	if oldPrice == nil || oldPrice.IsZero() {
		return false
	}
	if !newPrice.LessThan(*oldPrice) {
		return false
	}
	drop := newPrice.Sub(*oldPrice).Div(*oldPrice).Abs()
	return drop.GreaterThan(decimal.NewFromFloat(0.90))
}

func AddAllocation(buckets map[string]*models.AllocationRow, key, label string, value decimal.Decimal) {
	if bucket, ok := buckets[key]; ok {
		bucket.Value = bucket.Value.Add(value)
		return
	}
	buckets[key] = &models.AllocationRow{Key: key, Label: label, Value: value}
}

func AllocationRows(buckets map[string]*models.AllocationRow, total decimal.Decimal) []models.AllocationRow {
	rows := make([]models.AllocationRow, 0, len(buckets))
	for _, bucket := range buckets {
		if total.IsPositive() {
			bucket.Weight = bucket.Value.Div(total).Round(6)
		}
		rows = append(rows, *bucket)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].Value.Equal(rows[j].Value) {
			return rows[i].Value.GreaterThan(rows[j].Value)
		}
		return rows[i].Key < rows[j].Key
	})

	return rows
}

type CashFlow struct {
	Date   time.Time
	Amount decimal.Decimal
}

var (
	ErrXIRRNotEnoughFlows = errors.New("xirr needs at least two cash flows")
	ErrXIRRNoSignChange   = errors.New("xirr needs both a negative and a positive cash flow")
	ErrXIRRSpanTooShort   = errors.New("xirr needs a span of at least 30 days")
	ErrXIRRAboveRange     = errors.New("xirr is above the searchable range")
	ErrXIRRNoConvergence  = errors.New("xirr did not converge")
)

const (
	xirrMinSpanDays = 30
	xirrRateFloor   = -0.9999
	xirrRateCeil    = 100.0
	xirrIterations  = 200
	xirrDaysInYear  = 365.0
)

func XIRR(flows []CashFlow) (decimal.Decimal, error) {
	if len(flows) < 2 {
		return decimal.Zero, ErrXIRRNotEnoughFlows
	}

	sorted := make([]CashFlow, len(flows))
	copy(sorted, flows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	start := sorted[0].Date.UTC().Truncate(24 * time.Hour)
	end := sorted[len(sorted)-1].Date.UTC().Truncate(24 * time.Hour)

	if end.Sub(start).Hours()/24 < xirrMinSpanDays {
		return decimal.Zero, ErrXIRRSpanTooShort
	}

	amounts := make([]float64, len(sorted))
	years := make([]float64, len(sorted))
	hasNegative := false
	hasPositive := false

	for i, flow := range sorted {
		amounts[i] = flow.Amount.InexactFloat64()
		years[i] = flow.Date.UTC().Truncate(24*time.Hour).Sub(start).Hours() / 24 / xirrDaysInYear

		if flow.Amount.IsNegative() {
			hasNegative = true
		}
		if flow.Amount.IsPositive() {
			hasPositive = true
		}
	}

	if !hasNegative || !hasPositive {
		return decimal.Zero, ErrXIRRNoSignChange
	}

	npv := func(rate float64) float64 {
		total := 0.0
		for i := range amounts {
			total += amounts[i] / math.Pow(1+rate, years[i])
		}
		return total
	}

	low, high := xirrRateFloor, xirrRateCeil
	npvLow, npvHigh := npv(low), npv(high)

	if math.IsNaN(npvLow) || math.IsNaN(npvHigh) {
		return decimal.Zero, ErrXIRRNoConvergence
	}

	if npvLow > 0 && npvHigh > 0 {
		return decimal.Zero, ErrXIRRAboveRange
	}

	if npvLow*npvHigh > 0 {
		return decimal.Zero, ErrXIRRNoConvergence
	}

	for range xirrIterations {
		mid := (low + high) / 2
		npvMid := npv(mid)

		if npvMid == 0 {
			low, high = mid, mid
			break
		}

		if npvLow*npvMid < 0 {
			high = mid
		} else {
			low, npvLow = mid, npvMid
		}
	}

	rate := (low + high) / 2
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return decimal.Zero, ErrXIRRNoConvergence
	}

	return decimal.NewFromFloat(rate).Round(6), nil
}

func XIRRReason(err error) string {
	switch {
	case errors.Is(err, ErrXIRRSpanTooShort):
		return "held for less than 30 days"
	case errors.Is(err, ErrXIRRNotEnoughFlows), errors.Is(err, ErrXIRRNoSignChange):
		return "not enough cash flow history"
	case errors.Is(err, ErrXIRRAboveRange):
		return "above 10,000%/year"
	default:
		return "could not be calculated"
	}
}
