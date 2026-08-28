package utils

import (
	"sort"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
)

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
