package tests

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/finance"
)

// MockPriceFetcher returns hardcoded prices for tickers used in tests.
// Exchange rates are fixed at 1.1 for EUR→USD (and inverse for USD→EUR).
type MockPriceFetcher struct{}

var mockPrices = map[string]*finance.PriceData{
	"BTC-USD": {Symbol: "BTC-USD", Price: 50000.0, Currency: "USD", LastUpdate: 1700000000},
	"BTC-EUR": {Symbol: "BTC-EUR", Price: 45000.0, Currency: "EUR", LastUpdate: 1700000000},
	"IWDA.AS": {Symbol: "IWDA.AS", Price: 100.0, Currency: "EUR", LastUpdate: 1700000000},
}

func (m *MockPriceFetcher) GetAssetPrice(_ context.Context, ticker string, _ models.InvestmentType) (*finance.PriceData, error) {
	if data, ok := mockPrices[ticker]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: unknown ticker %q", ticker)
}

func (m *MockPriceFetcher) GetAssetPriceOnDate(_ context.Context, ticker string, _ models.InvestmentType, _ time.Time) (*finance.PriceData, error) {
	return m.GetAssetPrice(context.Background(), ticker, "")
}

func (m *MockPriceFetcher) GetAssetPriceRange(_ context.Context, ticker string, from, to time.Time) ([]finance.DatedPrice, error) {
	data, ok := mockPrices[ticker]
	if !ok {
		return nil, fmt.Errorf("mock: unknown ticker %q", ticker)
	}

	var prices []finance.DatedPrice
	for day := from.UTC().Truncate(24 * time.Hour); !day.After(to); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		prices = append(prices, finance.DatedPrice{Date: day, Price: data.Price, Currency: data.Currency})
	}
	return prices, nil
}

func (m *MockPriceFetcher) GetPricesForMultipleAssets(_ context.Context, assets []finance.AssetRequest) (map[string]*finance.PriceData, error) {
	result := make(map[string]*finance.PriceData, len(assets))
	for _, a := range assets {
		data, err := m.GetAssetPrice(context.Background(), a.Ticker, a.InvestmentType)
		if err != nil {
			result[a.Ticker] = &finance.PriceData{Symbol: a.Ticker, Error: err}
			continue
		}
		result[a.Ticker] = data
	}
	return result, nil
}

func (m *MockPriceFetcher) GetExchangeRate(_ context.Context, fromCurrency, toCurrency string) (float64, error) {
	return mockExchangeRate(fromCurrency, toCurrency)
}

func (m *MockPriceFetcher) GetExchangeRateOnDate(_ context.Context, fromCurrency, toCurrency string, _ time.Time) (float64, error) {
	return mockExchangeRate(fromCurrency, toCurrency)
}

func mockExchangeRate(from, to string) (float64, error) {
	if from == to {
		return 1.0, nil
	}
	pair := from + "-" + to
	rates := map[string]float64{
		"EUR-USD": 1.1,
		"USD-EUR": 1.0 / 1.1,
	}
	if r, ok := rates[pair]; ok {
		return r, nil
	}
	return 1.0, nil
}
