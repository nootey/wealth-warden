package prices

import "wealth-warden/internal/models"

type PriceData struct {
	Symbol     string
	Price      float64
	Currency   string
	LastUpdate int64 // Unix timestamp of when the price was last updated
	Error      error // Per-symbol error if fetch failed
}

type AssetRequest struct {
	Ticker         string
	InvestmentType models.InvestmentType
	Exchange       string // For stocks/ETFs
	Currency       string // For crypto
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

type quoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
			RegularMarketTime  int64   `json:"regularMarketTime"`
			Currency           string  `json:"currency"`
		} `json:"result"`
	} `json:"quoteResponse"`
}
