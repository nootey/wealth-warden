package finance

import (
	"context"
	"fmt"
	"wealth-warden/internal/repositories"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type CurrencyManager interface {
	ConvertInvestmentValueToAccountCurrency(ctx context.Context, tx *gorm.DB, accountID, userID int64, accountCurrency string) (total, negative decimal.Decimal, err error)
}

type CurrencyConverter struct {
	priceFetchClient PriceFetcher
	investmentRepo   repositories.InvestmentRepositoryInterface
}

func NewCurrencyManager(client PriceFetcher, investmentRepo *repositories.InvestmentRepository) *CurrencyConverter {
	return &CurrencyConverter{
		priceFetchClient: client,
		investmentRepo:   investmentRepo,
	}
}

func (c *CurrencyConverter) GetExchangeRate(ctx context.Context, from, to string) (decimal.Decimal, error) {
	if from == to {
		return decimal.NewFromFloat(1.0), nil
	}

	if c.priceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}

	rate, err := c.priceFetchClient.GetExchangeRate(ctx, from, to)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get exchange rate from %s to %s: %w", from, to, err)
	}

	return decimal.NewFromFloat(rate), nil
}

func (c *CurrencyConverter) ConvertInvestmentValueToAccountCurrency(ctx context.Context, tx *gorm.DB, accountID, userID int64, accountCurrency string) (total, negative decimal.Decimal, err error) {
	assets, err := c.investmentRepo.FindAssetsByAccountID(ctx, tx, accountID, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	total = decimal.Zero
	negative = decimal.Zero

	for _, asset := range assets {
		if asset.Quantity.IsZero() {
			continue
		}

		valueInAccountCurrency := asset.CurrentValue
		profitLossInAccountCurrency := asset.ProfitLoss

		if asset.Currency != accountCurrency {
			exchangeRate, err := c.GetExchangeRate(ctx, asset.Currency, accountCurrency)
			if err != nil {
				return decimal.Zero, decimal.Zero, err
			}
			valueInAccountCurrency = asset.CurrentValue.Mul(exchangeRate)
			profitLossInAccountCurrency = asset.ProfitLoss.Mul(exchangeRate)
		}

		// Check P&L in account currency
		if profitLossInAccountCurrency.LessThan(decimal.Zero) {
			negative = negative.Add(profitLossInAccountCurrency)
		}

		total = total.Add(valueInAccountCurrency)
	}

	return total, negative, nil
}
