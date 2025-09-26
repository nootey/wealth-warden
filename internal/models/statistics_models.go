package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type BasicAccountStats struct {
	UserID            int64           `json:"user_id"`
	AccountID         *int64          `json:"account_id,omitempty"` // nil => all accounts
	Currency          string          `json:"currency"`
	Year              int             `json:"year"`
	Inflow            decimal.Decimal `json:"inflow"`
	Outflow           decimal.Decimal `json:"outflow"`
	Net               decimal.Decimal `json:"net"`
	AvgMonthlyInflow  decimal.Decimal `json:"avg_monthly_inflow"`
	AvgMonthlyOutflow decimal.Decimal `json:"avg_monthly_outflow"`
	ActiveMonths      int             `json:"active_months"`
	Categories        []CategoryStat  `json:"categories,omitempty"`
	GeneratedAt       time.Time       `json:"generated_at"`
}

type CategoryStat struct {
	CategoryID   int64           `json:"category_id"`
	CategoryName *string         `json:"category_name,omitempty"`
	Inflow       decimal.Decimal `json:"inflow"`
	Outflow      decimal.Decimal `json:"outflow"`
	Net          decimal.Decimal `json:"net"`
	PctOfInflow  float64         `json:"pct_of_inflow"`
	PctOfOutflow float64         `json:"pct_of_outflow"`
}
