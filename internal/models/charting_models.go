package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type ChartPoint struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

type NetWorthResponse struct {
	Currency string       `json:"currency"`
	Points   []ChartPoint `json:"points"`
	Current  ChartPoint   `json:"current"`
}
