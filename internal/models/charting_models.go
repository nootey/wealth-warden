package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type ChartPoint struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

type Change struct {
	PrevPeriodEndDate  time.Time       `json:"prev_period_end_date"`
	PrevPeriodEndValue decimal.Decimal `json:"prev_period_end_value"`
	CurrentEndDate     time.Time       `json:"current_end_date"`
	CurrentEndValue    decimal.Decimal `json:"current_end_value"`
	Abs                decimal.Decimal `json:"abs"`
	Pct                decimal.Decimal `json:"pct"`
}

type NetWorthResponse struct {
	Currency string       `json:"currency"`
	Points   []ChartPoint `json:"points"`
	Current  ChartPoint   `json:"current"`
	Change   *Change      `json:"change,omitempty"`
}
