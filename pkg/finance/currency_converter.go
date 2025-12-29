package finance

import (
	"context"
	"wealth-warden/internal/repositories"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type CurrencyManager interface {
	ConvertInvestmentValueToAccountCurrency(ctx context.Context, tx *gorm.DB, accountID, userID int64, accountCurrency string) (decimal.Decimal, error)
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

func (c *CurrencyConverter) GetExchangeRate(ctx context.Context, from, to string) decimal.Decimal {
	if c.priceFetchClient != nil {
		rate, err := c.priceFetchClient.GetExchangeRate(ctx, from, to)
		if err == nil {
			return decimal.NewFromFloat(rate)
		}
	}
	return decimal.NewFromFloat(1.0)
}

func (c *CurrencyConverter) ConvertInvestmentValueToAccountCurrency(ctx context.Context, tx *gorm.DB, accountID, userID int64, accountCurrency string) (decimal.Decimal, error) {
	assets, err := c.investmentRepo.FindAssetsByAccountID(ctx, tx, accountID, userID)
	if err != nil {
		return decimal.Zero, err
	}

	totalValue := decimal.Zero
	for _, asset := range assets {
		if asset.Quantity.IsZero() {
			continue
		}

		valueInAccountCurrency := asset.CurrentValue
		if asset.Currency != accountCurrency {
			exchangeRate := c.GetExchangeRate(ctx, asset.Currency, accountCurrency)
			valueInAccountCurrency = asset.CurrentValue.Mul(exchangeRate)
		}
		totalValue = totalValue.Add(valueInAccountCurrency)
	}

	return totalValue, nil
}
