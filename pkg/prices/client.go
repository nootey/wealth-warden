package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"wealth-warden/internal/models"
)

type PriceFetcher interface {
	GetAssetPrice(ctx context.Context, ticker string, investmentType models.InvestmentType, opts ...string) (*PriceData, error)
	GetPricesForMultipleAssets(ctx context.Context, assets []AssetRequest) (map[string]*PriceData, error)
}

type PriceFetchClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewPriceFetchClient(baseURL string) (PriceFetcher, error) {

	if baseURL == "" {
		return nil, fmt.Errorf("base URL cannot be empty")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	if client.Timeout <= 0 {
		return nil, fmt.Errorf("invalid timeout configuration")
	}

	return &PriceFetchClient{
		httpClient: client,
		baseURL:    baseURL,
	}, nil
}

func (c *PriceFetchClient) normalizeExchange(exchange string) string {
	if exchange == "" {
		return ""
	}

	normalized := strings.ToUpper(strings.TrimSpace(exchange))

	// Map common exchange names to codes
	exchangeMap := map[string]string{
		"LONDON":    "L",
		"LSE":       "L",
		"AMSTERDAM": "AS",
		"EURONEXT":  "AS",
		"PARIS":     "PA",
		"GERMANY":   "DE",
		"XETRA":     "DE",
		"FRANKFURT": "F",
		"TORONTO":   "TO",
		"TSX":       "TO",
		"AUSTRALIA": "AX",
		"ASX":       "AX",
	}

	if code, exists := exchangeMap[normalized]; exists {
		return code
	}

	return normalized
}

func (c *PriceFetchClient) GetAssetPrice(ctx context.Context, ticker string, investmentType models.InvestmentType, opts ...string) (*PriceData, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	var exchange, currency string
	if len(opts) > 0 {
		exchange = c.normalizeExchange(opts[0])
	}
	if len(opts) > 1 {
		currency = strings.ToUpper(strings.TrimSpace(opts[1]))
	}

	var symbol string

	switch investmentType {
	case models.InvestmentCrypto:
		// Crypto: BTC-USD, ETH-USDT, etc.
		// Default to USDC if no currency provided
		if currency == "" {
			currency = "USD"
		}
		symbol = fmt.Sprintf("%s-%s", ticker, currency)

	case models.InvestmentStock, models.InvestmentETF:
		// Stock/ETF: exchange is required
		if exchange == "" {
			return nil, fmt.Errorf("exchange is required for stocks and ETFs")
		}
		symbol = fmt.Sprintf("%s.%s", ticker, exchange)

	default:
		return nil, fmt.Errorf("invalid investment type: %s", investmentType)
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", c.baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ticker '%s' not found on Yahoo Finance (status %d)", symbol, resp.StatusCode)
	}

	var data chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("no price data found for ticker '%s'", symbol)
	}

	meta := data.Chart.Result[0].Meta

	if meta.RegularMarketPrice == 0 {
		return nil, fmt.Errorf("invalid price (0) for ticker '%s'", symbol)
	}

	return &PriceData{
		Symbol:     meta.Symbol,
		Price:      meta.RegularMarketPrice,
		Currency:   meta.Currency,
		LastUpdate: meta.RegularMarketTime,
	}, nil
}

func (c *PriceFetchClient) GetPricesForMultipleAssets(ctx context.Context, assets []AssetRequest) (map[string]*PriceData, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("no assets provided")
	}

	symbols := make([]string, 0, len(assets))
	symbolMap := make(map[string]string) // maps yahoo symbol to original identifier

	for _, asset := range assets {
		ticker := strings.ToUpper(strings.TrimSpace(asset.Ticker))
		var symbol string
		var identifier string

		switch asset.InvestmentType {
		case models.InvestmentCrypto:
			currency := strings.ToUpper(strings.TrimSpace(asset.Currency))
			if currency == "" {
				currency = "USD"
			}
			symbol = fmt.Sprintf("%s-%s", ticker, currency)
			identifier = symbol

		case models.InvestmentStock, models.InvestmentETF:
			exchange := c.normalizeExchange(asset.Exchange)
			if exchange == "" {
				continue // Skip assets without exchange
			}
			symbol = fmt.Sprintf("%s.%s", ticker, exchange)
			identifier = symbol

		default:
			continue // Skip invalid types
		}

		symbols = append(symbols, symbol)
		symbolMap[symbol] = identifier
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("no valid assets to fetch")
	}

	symbolsParam := strings.Join(symbols, ",")
	url := fmt.Sprintf("%s/v7/finance/quote?symbols=%s", c.baseURL, symbolsParam)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d", resp.StatusCode)
	}

	var data quoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make(map[string]*PriceData)

	for _, quote := range data.QuoteResponse.Result {
		result[quote.Symbol] = &PriceData{
			Symbol:     quote.Symbol,
			Price:      quote.RegularMarketPrice,
			Currency:   quote.Currency,
			LastUpdate: quote.RegularMarketTime,
		}
	}

	// Mark missing symbols
	for _, symbol := range symbols {
		if _, exists := result[symbol]; !exists {
			result[symbol] = &PriceData{
				Symbol: symbol,
				Error:  fmt.Errorf("no data returned"),
			}
		}
	}

	return result, nil
}
