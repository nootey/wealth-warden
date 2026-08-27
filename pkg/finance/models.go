package finance

import (
	"time"
	"wealth-warden/internal/models"
)

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

type ChartMeta struct {
	Symbol             string  `json:"symbol"`
	Currency           string  `json:"currency"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	RegularMarketTime  int64   `json:"regularMarketTime"`
	GMTOffset          int64   `json:"gmtoffset"`
}

type ChartQuote struct {
	Close []*float64 `json:"close"`
}

type ChartResult struct {
	Meta       ChartMeta `json:"meta"`
	Timestamp  []int64   `json:"timestamp"`
	Indicators struct {
		Quote []ChartQuote `json:"quote"`
	} `json:"indicators"`
}

type ChartResponse struct {
	Chart struct {
		Result []ChartResult `json:"result"`
	} `json:"chart"`
}

type DatedPrice struct {
	Date     time.Time
	Price    float64
	Currency string
}

type QuoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
			RegularMarketTime  int64   `json:"regularMarketTime"`
			Currency           string  `json:"currency"`
		} `json:"result"`
	} `json:"quoteResponse"`
}
