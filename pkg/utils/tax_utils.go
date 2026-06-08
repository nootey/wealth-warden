package utils

import (
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
)

// ApplyBracket returns the applicable bracket for daysHeld using a step-function.
// brackets must be sorted by MinDaysHeld ASC (repo guarantees this).
func ApplyBracket(brackets []models.InvestmentTaxBracket, daysHeld int) *models.InvestmentTaxBracket {
	var result *models.InvestmentTaxBracket
	for i := range brackets {
		if daysHeld >= brackets[i].MinDaysHeld {
			result = &brackets[i]
		}
	}
	return result
}

type FifoLot struct {
	TxnDate    time.Time
	Quantity   decimal.Decimal
	ValueAtBuy decimal.Decimal
	Fee        decimal.Decimal
}

// BuildFifoLots processes trades in order (txn_date ASC, id ASC), consuming sells
// against buy lots, and returns the remaining open lots. If stopAtID > 0 the loop
// halts before that trade ID — use this when computing hold-period for a sell.
func BuildFifoLots(trades []models.InvestmentTrade, stopAtID int64) []FifoLot {
	var lots []FifoLot
	for _, t := range trades {
		if stopAtID > 0 && t.ID == stopAtID {
			break
		}
		if t.TradeType == models.InvestmentBuy {
			lots = append(lots, FifoLot{
				TxnDate:    t.TxnDate,
				Quantity:   t.Quantity,
				ValueAtBuy: t.ValueAtBuy,
				Fee:        t.Fee,
			})
		} else {
			toConsume := t.Quantity
			for i := range lots {
				if toConsume.IsZero() {
					break
				}
				if lots[i].Quantity.LessThanOrEqual(toConsume) {
					toConsume = toConsume.Sub(lots[i].Quantity)
					lots[i].Quantity = decimal.Zero
					lots[i].ValueAtBuy = decimal.Zero
					lots[i].Fee = decimal.Zero
				} else {
					proportion := toConsume.Div(lots[i].Quantity)
					lots[i].ValueAtBuy = lots[i].ValueAtBuy.Sub(lots[i].ValueAtBuy.Mul(proportion))
					lots[i].Fee = lots[i].Fee.Sub(lots[i].Fee.Mul(proportion))
					lots[i].Quantity = lots[i].Quantity.Sub(toConsume)
					toConsume = decimal.Zero
				}
			}
		}
	}
	return lots
}

// ComputeBuyTradeTaxInfo returns tax info for an open (unrealized) buy trade.
func ComputeBuyTradeTaxInfo(trade models.InvestmentTrade, brackets []models.InvestmentTaxBracket, today time.Time) models.TradeTaxInfo {
	daysHeld := int(today.UTC().Sub(trade.TxnDate.UTC()) / (24 * time.Hour))
	info := models.TradeTaxInfo{DaysHeld: daysHeld}

	bracket := ApplyBracket(brackets, daysHeld)
	if bracket != nil {
		p := bracket.TaxablePercent
		info.TaxablePercent = &p
		taxDue := decimal.Zero
		if trade.ProfitLoss.IsPositive() {
			taxDue = trade.ProfitLoss.Mul(bracket.TaxablePercent).Div(decimal.NewFromInt(100))
		}
		info.TaxableProfit = trade.ProfitLoss.Sub(taxDue)
	}

	for _, b := range brackets {
		if b.MinDaysHeld > daysHeld {
			rem := b.MinDaysHeld - daysHeld
			info.DaysUntilNextBracket = &rem
			break
		}
	}

	if bracket != nil && bracket.TaxablePercent.IsZero() {
		zero := 0
		info.DaysUntilTaxFree = &zero
	} else {
		for _, b := range brackets {
			if b.TaxablePercent.IsZero() && b.MinDaysHeld > daysHeld {
				rem := b.MinDaysHeld - daysHeld
				info.DaysUntilTaxFree = &rem
				break
			}
		}
	}

	return info
}

// ComputeSellTradeTaxInfo returns tax info for a realized sell trade.
// allAssetTrades must include all trades for the asset sorted (txn_date ASC, id ASC).
func ComputeSellTradeTaxInfo(sell models.InvestmentTrade, allAssetTrades []models.InvestmentTrade, brackets []models.InvestmentTaxBracket) models.TradeTaxInfo {
	lots := BuildFifoLots(allAssetTrades, sell.ID)

	remaining := sell.Quantity
	weightedDays := decimal.Zero
	consumed := decimal.Zero

	for _, lot := range lots {
		if lot.Quantity.IsZero() || remaining.IsZero() {
			continue
		}
		c := decimal.Min(lot.Quantity, remaining)
		days := int64(sell.TxnDate.UTC().Sub(lot.TxnDate.UTC()) / (24 * time.Hour))
		weightedDays = weightedDays.Add(c.Mul(decimal.NewFromInt(days)))
		consumed = consumed.Add(c)
		remaining = remaining.Sub(c)
	}

	var daysHeld int
	if consumed.IsPositive() {
		daysHeld = int(weightedDays.Div(consumed).IntPart())
	}

	info := models.TradeTaxInfo{DaysHeld: daysHeld}

	bracket := ApplyBracket(brackets, daysHeld)
	if bracket != nil {
		p := bracket.TaxablePercent
		info.TaxablePercent = &p
		taxDue := decimal.Zero
		if sell.ProfitLoss.IsPositive() {
			taxDue = sell.ProfitLoss.Mul(bracket.TaxablePercent).Div(decimal.NewFromInt(100))
		}
		info.TaxableProfit = sell.ProfitLoss.Sub(taxDue)
	}

	return info
}

// ComputeAssetTaxSummary computes after-tax PnL for an asset across all open lots.
// allAssetTrades must be sorted (txn_date ASC, id ASC).
func ComputeAssetTaxSummary(asset models.InvestmentAsset, allAssetTrades []models.InvestmentTrade, brackets []models.InvestmentTaxBracket, settings models.InvestmentTaxSettings, today time.Time) models.AssetTaxSummary {
	if len(brackets) == 0 || asset.CurrentPrice == nil || asset.CurrentPrice.IsZero() {
		return models.AssetTaxSummary{AfterTaxPnL: asset.ProfitLoss}
	}

	lots := BuildFifoLots(allAssetTrades, 0)

	type openLot struct {
		txnDate   time.Time
		quantity  decimal.Decimal
		costBasis decimal.Decimal
	}

	var openLots []openLot
	for _, lot := range lots {
		if lot.Quantity.IsZero() {
			continue
		}
		cb := lot.ValueAtBuy
		if asset.InvestmentType != models.InvestmentCrypto {
			cb = cb.Add(lot.Fee)
		}
		openLots = append(openLots, openLot{txnDate: lot.TxnDate, quantity: lot.Quantity, costBasis: cb})
	}

	if !settings.LossOffsettingEnabled {
		totalTax := decimal.Zero
		for _, lot := range openLots {
			daysHeld := int(today.UTC().Sub(lot.txnDate.UTC()) / (24 * time.Hour))
			bracket := ApplyBracket(brackets, daysHeld)
			if bracket == nil {
				continue
			}
			pnl := lot.quantity.Mul(*asset.CurrentPrice).Sub(lot.costBasis)
			if pnl.IsPositive() {
				totalTax = totalTax.Add(pnl.Mul(bracket.TaxablePercent).Div(decimal.NewFromInt(100)))
			}
		}
		return models.AssetTaxSummary{
			EstimatedTaxDue: totalTax,
			AfterTaxPnL:     asset.ProfitLoss.Sub(totalTax),
		}
	}

	totalPnL := decimal.Zero
	weightedDays := decimal.Zero
	totalQty := decimal.Zero
	for _, lot := range openLots {
		daysHeld := int(today.UTC().Sub(lot.txnDate.UTC()) / (24 * time.Hour))
		pnl := lot.quantity.Mul(*asset.CurrentPrice).Sub(lot.costBasis)
		totalPnL = totalPnL.Add(pnl)
		weightedDays = weightedDays.Add(lot.quantity.Mul(decimal.NewFromInt(int64(daysHeld))))
		totalQty = totalQty.Add(lot.quantity)
	}

	var totalTax decimal.Decimal
	if totalPnL.IsPositive() && totalQty.IsPositive() {
		avgDays := int(weightedDays.Div(totalQty).IntPart())
		bracket := ApplyBracket(brackets, avgDays)
		if bracket != nil {
			totalTax = totalPnL.Mul(bracket.TaxablePercent).Div(decimal.NewFromInt(100))
		}
	}

	return models.AssetTaxSummary{
		EstimatedTaxDue: totalTax,
		AfterTaxPnL:     asset.ProfitLoss.Sub(totalTax),
	}
}
