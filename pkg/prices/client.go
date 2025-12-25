package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type PriceFetcher interface {
	GetAssetPrice(ctx context.Context, ticker string, investmentType models.InvestmentType) (*PriceData, error)
	GetAssetPriceOnDate(ctx context.Context, ticker string, investmentType models.InvestmentType, date time.Time) (*PriceData, error)
	GetPricesForMultipleAssets(ctx context.Context, assets []AssetRequest) (map[string]*PriceData, error)
	GetExchangeRate(ctx context.Context, currency string) (float64, error)
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

func (c *PriceFetchClient) GetAssetPrice(ctx context.Context, ticker string, investmentType models.InvestmentType) (*PriceData, error) {

	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	query := "interval=1d&range=1d"
	if investmentType == models.InvestmentStock || investmentType == models.InvestmentETF {
		query = "interval=1d&range=7d"
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?%s", c.baseURL, ticker, query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ticker '%s' not found on Yahoo Finance (status %d)", ticker, resp.StatusCode)
	}

	var data ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("no price data found for ticker '%s'", ticker)
	}

	result := data.Chart.Result[0]
	meta := result.Meta

	// Try to pick the most recent valid close from the candles (most recent non-weekend day)
	// Should also handle holidays
	var (
		bestPrice float64
		bestTime  int64
	)

	if result.Timestamp != nil &&
		result.Indicators.Quote != nil && len(result.Indicators.Quote) > 0 &&
		result.Indicators.Quote[0].Close != nil {

		closes := result.Indicators.Quote[0].Close
		for i := len(result.Timestamp) - 1; i >= 0; i-- {
			if i < len(closes) && closes[i] != nil {
				p := *closes[i]
				if p > 0 {
					bestPrice = p
					bestTime = result.Timestamp[i]
					break
				}
			}
		}
	}

	// Fallback: meta regular market price (might be 0 on weekends)
	if bestPrice == 0 && meta.RegularMarketPrice > 0 {
		bestPrice = meta.RegularMarketPrice
		bestTime = meta.RegularMarketTime
	}

	if bestPrice == 0 {
		return nil, fmt.Errorf("no valid recent price found for ticker '%s'", ticker)
	}

	return &PriceData{
		Symbol:     meta.Symbol,
		Price:      bestPrice,
		Currency:   meta.Currency,
		LastUpdate: bestTime,
	}, nil
}

func (c *PriceFetchClient) GetAssetPriceOnDate(ctx context.Context, ticker string, investmentType models.InvestmentType, date time.Time) (*PriceData, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	if investmentType == models.InvestmentStock || investmentType == models.InvestmentETF {
		date = utils.AdjustToWeekday(date)
	}

	// Convert date to Unix timestamp
	// Yahoo needs period1 (start) and period2 (end)
	// For a specific date, query that day + 1 day buffer
	startOfDay := date.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.AddDate(0, 0, 2) // Add 2 days buffer

	period1 := startOfDay.Unix()
	period2 := endOfDay.Unix()

	url := fmt.Sprintf("%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		c.baseURL, ticker, period1, period2)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ticker '%s' not found on Yahoo Finance (status %d)", ticker, resp.StatusCode)
	}

	var data ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("no price data found for ticker '%s'", ticker)
	}

	result := data.Chart.Result[0]

	if len(result.Timestamp) == 0 {
		return nil, fmt.Errorf("no historical data available for %s", ticker)
	}

	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data available for %s", ticker)
	}

	quotes := result.Indicators.Quote[0]
	if len(quotes.Close) == 0 {
		return nil, fmt.Errorf("no close prices available for %s", ticker)
	}

	var closestPrice float64
	var closestTime int64

	for i, ts := range result.Timestamp {
		if i < len(quotes.Close) && quotes.Close[i] != nil {
			closestPrice = *quotes.Close[i]
			closestTime = ts
			break
		}
	}

	if closestPrice == 0 {
		return nil, fmt.Errorf("no valid price found for %s on %s", ticker, date.Format("2006-01-02"))
	}

	return &PriceData{
		Symbol:     result.Meta.Symbol,
		Price:      closestPrice,
		Currency:   result.Meta.Currency,
		LastUpdate: closestTime,
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
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d", resp.StatusCode)
	}

	var data QuoteResponse
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

func (c *PriceFetchClient) GetExchangeRate(ctx context.Context, currency string) (float64, error) {
	if strings.ToUpper(currency) == "USD" {
		return 1.0, nil
	}

	// Yahoo format: EUR=X for EUR/USD rate
	symbol := fmt.Sprintf("%s=X", strings.ToUpper(currency))

	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", c.baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate: %w", err)
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get exchange rate for %s (status %d)", currency, resp.StatusCode)
	}

	var data ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data.Chart.Result) == 0 {
		return 0, fmt.Errorf("no exchange rate data for %s", currency)
	}

	rate := data.Chart.Result[0].Meta.RegularMarketPrice
	if rate == 0 {
		return 0, fmt.Errorf("invalid exchange rate (0) for %s", currency)
	}

	return rate, nil
}
