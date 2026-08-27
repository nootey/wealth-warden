package finance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/finance"
)

var (
	testServer        *httptest.Server
	fetcher           finance.PriceFetcher
	mockResponse      *finance.ChartResponse
	mockQuoteResponse *finance.QuoteResponse
	mockStatusCode    int
)

func setupTestServer() {
	testServer = httptest.NewServer(http.HandlerFunc(handleTestRequest))
	var err error
	fetcher, err = finance.NewPriceFetchClient(testServer.URL)
	if err != nil {
		panic("Failed to create price fetcher: " + err.Error())
	}
}

func teardownTestServer() {
	if testServer != nil {
		testServer.Close()
	}
}

func handleTestRequest(w http.ResponseWriter, r *http.Request) {
	if mockStatusCode != 0 {
		w.WriteHeader(mockStatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if mockQuoteResponse != nil {
		err := json.NewEncoder(w).Encode(mockQuoteResponse)
		if err != nil {
			return
		}
	} else {
		err := json.NewEncoder(w).Encode(mockResponse)
		if err != nil {
			return
		}
	}
}

func setupChartResponse(symbol, currency string, price float64, timestamp int64) {
	mockStatusCode = 0
	response := finance.ChartResponse{}
	response.Chart.Result = make([]finance.ChartResult, 1)

	response.Chart.Result[0].Meta.Symbol = symbol
	response.Chart.Result[0].Meta.Currency = currency
	response.Chart.Result[0].Meta.RegularMarketPrice = price
	response.Chart.Result[0].Meta.RegularMarketTime = timestamp
	response.Chart.Result[0].Timestamp = []int64{timestamp}
	response.Chart.Result[0].Indicators.Quote = make([]finance.ChartQuote, 1)
	response.Chart.Result[0].Indicators.Quote[0].Close = []*float64{&price}

	mockResponse = &response
}

func setupQuoteResponse(quotes map[string]struct {
	Price     float64
	Currency  string
	Timestamp int64
}) {
	mockStatusCode = 0
	response := finance.QuoteResponse{}
	response.QuoteResponse.Result = make([]struct {
		Symbol             string  `json:"symbol"`
		RegularMarketPrice float64 `json:"regularMarketPrice"`
		RegularMarketTime  int64   `json:"regularMarketTime"`
		Currency           string  `json:"currency"`
	}, 0)

	for symbol, data := range quotes {
		quote := struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
			RegularMarketTime  int64   `json:"regularMarketTime"`
			Currency           string  `json:"currency"`
		}{
			Symbol:             symbol,
			RegularMarketPrice: data.Price,
			RegularMarketTime:  data.Timestamp,
			Currency:           data.Currency,
		}
		response.QuoteResponse.Result = append(response.QuoteResponse.Result, quote)
	}

	mockQuoteResponse = &response
}

func resetMocks() {
	mockResponse = nil
	mockQuoteResponse = nil
	mockStatusCode = 0
}

func TestMain(m *testing.M) {
	setupTestServer()
	code := m.Run()
	teardownTestServer()
	os.Exit(code)
}

func TestGetAssetPrice_ValidStockTicker(t *testing.T) {
	resetMocks()
	timestamp := time.Now().Unix()
	setupChartResponse("AAPL", "USD", 150.25, timestamp)

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "AAPL", models.InvestmentStock)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", priceData.Symbol)
	}

	if priceData.Price != 150.25 {
		t.Errorf("Expected price 150.25, got %f", priceData.Price)
	}

	if priceData.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", priceData.Currency)
	}

	if priceData.LastUpdate == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestGetAssetPrice_ValidCryptoTicker(t *testing.T) {
	resetMocks()
	timestamp := time.Now().Unix()
	setupChartResponse("BTC-USD", "USD", 45000.50, timestamp)

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "BTC-USD", models.InvestmentCrypto)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Symbol != "BTC-USD" {
		t.Errorf("Expected symbol BTC-USD, got %s", priceData.Symbol)
	}

	if priceData.Price != 45000.50 {
		t.Errorf("Expected price 45000.50, got %f", priceData.Price)
	}

	if priceData.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", priceData.Currency)
	}

	if priceData.LastUpdate == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestGetAssetPrice_InvalidTicker(t *testing.T) {
	resetMocks()

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "INVALID", models.InvestmentStock)

	if err == nil {
		t.Fatal("Expected error for invalid ticker, got nil")
	}

	if priceData != nil {
		t.Errorf("Expected nil price data, got %+v", priceData)
	}
}

func TestGetAssetPrice_MalformedResponse(t *testing.T) {
	resetMocks()

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "AAPL", models.InvestmentStock)

	if err == nil {
		t.Fatal("Expected error for malformed response, got nil")
	}

	if priceData != nil {
		t.Errorf("Expected nil price data, got %+v", priceData)
	}
}

func TestGetAssetPrice_WeekendHandling(t *testing.T) {
	resetMocks()

	fridayTimestamp := time.Date(2024, 1, 5, 16, 0, 0, 0, time.UTC).Unix() // Friday close
	fridayPrice := 150.25

	response := finance.ChartResponse{}
	response.Chart.Result = make([]finance.ChartResult, 1)

	response.Chart.Result[0].Meta.Symbol = "AAPL"
	response.Chart.Result[0].Meta.Currency = "USD"
	response.Chart.Result[0].Meta.RegularMarketPrice = 0 // Weekend - market closed
	response.Chart.Result[0].Meta.RegularMarketTime = time.Now().Unix()
	response.Chart.Result[0].Timestamp = []int64{fridayTimestamp}
	response.Chart.Result[0].Indicators.Quote = make([]finance.ChartQuote, 1)
	response.Chart.Result[0].Indicators.Quote[0].Close = []*float64{&fridayPrice}

	mockStatusCode = 0
	mockResponse = &response

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "AAPL", models.InvestmentStock)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Price != fridayPrice {
		t.Errorf("Expected Friday's price %f, got %f", fridayPrice, priceData.Price)
	}

	if priceData.LastUpdate != fridayTimestamp {
		t.Errorf("Expected Friday's timestamp, got different timestamp")
	}
}

func TestGetAssetPriceOnDate_ValidHistoricalDate(t *testing.T) {
	resetMocks()

	historicalDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	timestamp := historicalDate.Unix()
	setupChartResponse("AAPL", "USD", 185.50, timestamp)

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPriceOnDate(ctx, "AAPL", models.InvestmentStock, historicalDate)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", priceData.Symbol)
	}

	if priceData.Price != 185.50 {
		t.Errorf("Expected price 185.50, got %f", priceData.Price)
	}

	if priceData.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", priceData.Currency)
	}

	if priceData.LastUpdate != timestamp {
		t.Errorf("Expected timestamp %d, got %d", timestamp, priceData.LastUpdate)
	}
}

func TestGetAssetPriceOnDate_WeekendDateAdjusted(t *testing.T) {
	resetMocks()

	saturdayDate := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC) // Saturday
	fridayDate := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)   // Friday
	fridayTimestamp := fridayDate.Unix()

	setupChartResponse("AAPL", "USD", 175.30, fridayTimestamp)

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPriceOnDate(ctx, "AAPL", models.InvestmentStock, saturdayDate)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Price != 175.30 {
		t.Errorf("Expected price 175.30, got %f", priceData.Price)
	}

	if priceData.LastUpdate != fridayTimestamp {
		t.Errorf("Expected Friday's timestamp, got %d", priceData.LastUpdate)
	}
}

func TestGetAssetPriceOnDate_NoDataAvailable(t *testing.T) {
	resetMocks()

	historicalDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	response := finance.ChartResponse{}
	response.Chart.Result = make([]finance.ChartResult, 1)

	response.Chart.Result[0].Timestamp = []int64{}
	response.Chart.Result[0].Indicators.Quote = make([]finance.ChartQuote, 1)
	response.Chart.Result[0].Indicators.Quote[0].Close = []*float64{}

	mockStatusCode = 0
	mockResponse = &response

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPriceOnDate(ctx, "AAPL", models.InvestmentStock, historicalDate)

	if err == nil {
		t.Fatal("Expected error for no data available, got nil")
	}

	if priceData != nil {
		t.Errorf("Expected nil price data, got %+v", priceData)
	}
}

func TestGetAssetPriceOnDate_FutureDate(t *testing.T) {
	resetMocks()
	futureDate := time.Now().AddDate(1, 0, 0) // 1 year in the future
	todayTimestamp := time.Now().Unix()

	setupChartResponse("AAPL", "USD", 150.25, todayTimestamp)

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPriceOnDate(ctx, "AAPL", models.InvestmentStock, futureDate)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if priceData.Price != 150.25 {
		t.Errorf("Expected price 150.25, got %f", priceData.Price)
	}
}

func TestGetPricesForMultipleAssets_ValidTickers(t *testing.T) {
	resetMocks()

	timestamp := time.Now().Unix()
	setupQuoteResponse(map[string]struct {
		Price     float64
		Currency  string
		Timestamp int64
	}{
		"AAPL.L": {Price: 150.25, Currency: "GBP", Timestamp: timestamp},
		"MSFT.L": {Price: 380.50, Currency: "GBP", Timestamp: timestamp},
	})

	ctx := context.Background()
	assets := []finance.AssetRequest{
		{Ticker: "AAPL", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
		{Ticker: "MSFT", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
	}

	results, err := fetcher.GetPricesForMultipleAssets(ctx, assets)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["AAPL.L"].Price != 150.25 {
		t.Errorf("Expected AAPL price 150.25, got %f", results["AAPL.L"].Price)
	}

	if results["MSFT.L"].Price != 380.50 {
		t.Errorf("Expected MSFT price 380.50, got %f", results["MSFT.L"].Price)
	}
}

func TestGetPricesForMultipleAssets_MixedValidInvalid(t *testing.T) {
	resetMocks()

	timestamp := time.Now().Unix()
	setupQuoteResponse(map[string]struct {
		Price     float64
		Currency  string
		Timestamp int64
	}{
		"AAPL.L": {Price: 150.25, Currency: "GBP", Timestamp: timestamp},
	})

	ctx := context.Background()
	assets := []finance.AssetRequest{
		{Ticker: "AAPL", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
		{Ticker: "INVALID", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
	}

	results, err := fetcher.GetPricesForMultipleAssets(ctx, assets)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["AAPL.L"].Price != 150.25 {
		t.Errorf("Expected AAPL price 150.25, got %f", results["AAPL.L"].Price)
	}

	if results["INVALID.L"].Error == nil {
		t.Error("Expected error for invalid ticker, got nil")
	}
}

func TestGetPricesForMultipleAssets_EmptyList(t *testing.T) {
	resetMocks()

	ctx := context.Background()
	var assets []finance.AssetRequest

	results, err := fetcher.GetPricesForMultipleAssets(ctx, assets)

	if err == nil {
		t.Fatal("Expected error for empty asset list, got nil")
	}

	if results != nil {
		t.Errorf("Expected nil results, got %+v", results)
	}
}

func TestGetPricesForMultipleAssets_DifferentTypes(t *testing.T) {
	resetMocks()

	timestamp := time.Now().Unix()
	setupQuoteResponse(map[string]struct {
		Price     float64
		Currency  string
		Timestamp int64
	}{
		"AAPL.L":  {Price: 150.25, Currency: "GBP", Timestamp: timestamp},
		"BTC-USD": {Price: 45000.50, Currency: "USD", Timestamp: timestamp},
	})

	ctx := context.Background()
	assets := []finance.AssetRequest{
		{Ticker: "AAPL", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
		{Ticker: "BTC", InvestmentType: models.InvestmentCrypto, Currency: "USD"},
	}

	results, err := fetcher.GetPricesForMultipleAssets(ctx, assets)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["AAPL.L"].Price != 150.25 {
		t.Errorf("Expected AAPL price 150.25, got %f", results["AAPL.L"].Price)
	}

	if results["BTC-USD"].Price != 45000.50 {
		t.Errorf("Expected BTC price 45000.50, got %f", results["BTC-USD"].Price)
	}

	if results["AAPL.L"].Currency != "GBP" {
		t.Errorf("Expected GBP currency, got %s", results["AAPL.L"].Currency)
	}

	if results["BTC-USD"].Currency != "USD" {
		t.Errorf("Expected USD currency, got %s", results["BTC-USD"].Currency)
	}
}

func TestGetExchangeRate_USD(t *testing.T) {
	resetMocks()

	ctx := context.Background()
	rate, err := fetcher.GetExchangeRate(ctx, "USD", "USD")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if rate != 1.0 {
		t.Errorf("Expected rate 1.0 for USD, got %f", rate)
	}
}

func TestGetExchangeRate_ValidCurrency(t *testing.T) {
	resetMocks()

	timestamp := time.Now().Unix()
	setupChartResponse("EUR=X", "USD", 1.18, timestamp)
	mockQuoteResponse = nil

	ctx := context.Background()
	rate, err := fetcher.GetExchangeRate(ctx, "EUR", "USD")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if rate != 1.18 {
		t.Errorf("Expected rate 1.18, got %f", rate)
	}
}

func TestGetExchangeRate_InvalidCurrency(t *testing.T) {
	resetMocks()

	mockStatusCode = http.StatusNotFound
	mockResponse = nil

	ctx := context.Background()
	rate, err := fetcher.GetExchangeRate(ctx, "INVALID", "USD")

	if err == nil {
		t.Fatal("Expected error for invalid currency, got nil")
	}

	if rate != 0 {
		t.Errorf("Expected rate 0, got %f", rate)
	}
}

func TestGetAssetPrice_EmptyTicker(t *testing.T) {
	resetMocks()

	ctx := context.Background()
	priceData, err := fetcher.GetAssetPrice(ctx, "  ", models.InvestmentStock)

	if err == nil {
		t.Fatal("Expected error for empty ticker, got nil")
	}

	if priceData != nil {
		t.Errorf("Expected nil price data, got %+v", priceData)
	}
}

func TestGetPricesForMultipleAssets_ExchangeNormalization(t *testing.T) {
	resetMocks()

	timestamp := time.Now().Unix()
	setupQuoteResponse(map[string]struct {
		Price     float64
		Currency  string
		Timestamp int64
	}{
		"AAPL.L": {Price: 150.25, Currency: "GBP", Timestamp: timestamp},
		"MSFT.L": {Price: 380.50, Currency: "GBP", Timestamp: timestamp},
	})

	ctx := context.Background()
	assets := []finance.AssetRequest{
		{Ticker: "AAPL", InvestmentType: models.InvestmentStock, Exchange: "LSE"},
		{Ticker: "MSFT", InvestmentType: models.InvestmentStock, Exchange: "LONDON"},
	}

	results, err := fetcher.GetPricesForMultipleAssets(ctx, assets)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["AAPL.L"] == nil {
		t.Error("Expected AAPL.L result")
	}

	if results["MSFT.L"] == nil {
		t.Error("Expected MSFT.L result (LONDON should normalize to L)")
	}
}

func TestNewPriceFetchClient_EmptyBaseURL(t *testing.T) {
	resetMocks()

	_, err := finance.NewPriceFetchClient("")

	if err == nil {
		t.Fatal("Expected error for empty base URL, got nil")
	}
}

func setupChartSeries(symbol, currency string, gmtOffset int64, timestamps []int64, closes []*float64) {
	mockStatusCode = 0
	response := finance.ChartResponse{}
	response.Chart.Result = make([]finance.ChartResult, 1)
	response.Chart.Result[0].Meta.Symbol = symbol
	response.Chart.Result[0].Meta.Currency = currency
	response.Chart.Result[0].Meta.GMTOffset = gmtOffset
	response.Chart.Result[0].Timestamp = timestamps
	response.Chart.Result[0].Indicators.Quote = make([]finance.ChartQuote, 1)
	response.Chart.Result[0].Indicators.Quote[0].Close = closes

	mockResponse = &response
}

func TestGetAssetPriceRange_SkipsGapsAndDaysOutsideRange(t *testing.T) {
	resetMocks()

	// Amsterdam opens 09:00 local, which is 07:00 UTC in summer
	session := func(day int) int64 {
		return time.Date(2026, 8, day, 7, 0, 0, 0, time.UTC).Unix()
	}
	monday := 100.0
	wednesday := 102.0
	thursday := 103.0

	setupChartSeries("IWDA.AS", "EUR", 7200,
		[]int64{session(24), session(25), session(26), session(27)},
		[]*float64{&monday, nil, &wednesday, &thursday})

	from := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)

	prices, err := fetcher.GetAssetPriceRange(context.Background(), "IWDA.AS", from, to)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("Expected 2 prices, got %d: %+v", len(prices), prices)
	}

	if !prices[0].Date.Equal(from) || prices[0].Price != monday || prices[0].Currency != "EUR" {
		t.Errorf("Unexpected first price: %+v", prices[0])
	}

	if !prices[1].Date.Equal(to) || prices[1].Price != wednesday {
		t.Errorf("Unexpected second price: %+v", prices[1])
	}
}

func TestGetAssetPriceRange_ClosedMarketIsNotAnError(t *testing.T) {
	resetMocks()
	setupChartSeries("IWDA.AS", "EUR", 7200, []int64{}, []*float64{})

	christmas := time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)

	prices, err := fetcher.GetAssetPriceRange(context.Background(), "IWDA.AS", christmas, christmas)
	if err != nil {
		t.Fatalf("Expected no error for a closed market, got: %v", err)
	}

	if len(prices) != 0 {
		t.Errorf("Expected no prices, got %+v", prices)
	}
}
