package jobqueue

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
)

type RecalculateAssetPnLJob struct {
	Repo             repositories.InvestmentRepositoryInterface
	AccRepo          repositories.AccountRepositoryInterface
	PriceFetchClient finance.PriceFetcher
	AssetID          int64
	UserID           int64
}

func (j *RecalculateAssetPnLJob) Process(ctx context.Context) error {
	tx, err := j.Repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	asset, err := j.Repo.FindInvestmentAssetByID(ctx, tx, j.AssetID, j.UserID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Fetch fresh price
	var currentPrice *decimal.Decimal
	var lastPriceUpdate *time.Time

	if j.PriceFetchClient != nil {
		priceData, err := j.PriceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err == nil && priceData != nil && priceData.Price > 0 {
			price := decimal.NewFromFloat(priceData.Price)
			now := time.Unix(priceData.LastUpdate, 0)
			currentPrice = &price
			lastPriceUpdate = &now
		}
	}

	// If price could not be fetched, keep the existing one
	if currentPrice == nil {
		currentPrice = asset.CurrentPrice
		lastPriceUpdate = asset.LastPriceUpdate
	}

	trades, err := j.Repo.FindInvestmentTradesByAssetID(ctx, tx, j.AssetID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find trades: %w", err)
	}

	var totalQuantity decimal.Decimal
	var totalCost decimal.Decimal

	for _, trade := range trades {
		switch trade.TradeType {
		case models.InvestmentBuy:
			totalQuantity = totalQuantity.Add(trade.Quantity)
			totalCost = totalCost.Add(trade.ValueAtBuy)
		case models.InvestmentSell:
			totalQuantity = totalQuantity.Sub(trade.Quantity)
			if !totalQuantity.IsZero() {
				avgPrice := totalCost.Div(totalQuantity.Add(trade.Quantity))
				totalCost = totalQuantity.Mul(avgPrice)
			}
		}
	}

	var newAvgBuyPrice decimal.Decimal
	if !totalQuantity.IsZero() {
		newAvgBuyPrice = totalCost.Div(totalQuantity)
	}

	var newCurrentValue, newProfitLoss, newProfitLossPercent decimal.Decimal
	if currentPrice != nil && !currentPrice.IsZero() {
		newCurrentValue = totalQuantity.Mul(*currentPrice)
		newProfitLoss = newCurrentValue.Sub(totalCost)
		if !totalCost.IsZero() {
			newProfitLossPercent = newProfitLoss.Div(totalCost)
		}
	}

	updates := map[string]interface{}{
		"quantity":            totalQuantity,
		"average_buy_price":   newAvgBuyPrice,
		"value_at_buy":        totalCost,
		"current_value":       newCurrentValue,
		"profit_loss":         newProfitLoss,
		"profit_loss_percent": newProfitLossPercent,
		"updated_at":          time.Now(),
	}

	if currentPrice != nil {
		updates["current_price"] = *currentPrice
	}
	if lastPriceUpdate != nil {
		updates["last_price_update"] = *lastPriceUpdate
	}

	err = tx.Model(&models.InvestmentAsset{}).
		Where("id = ?", j.AssetID).
		Updates(updates).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update asset: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}
