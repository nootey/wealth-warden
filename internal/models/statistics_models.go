package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type BasicAccountStats struct {
	UserID             int64           `json:"user_id"`
	AccountID          *int64          `json:"account_id,omitempty"` // nil => all accounts
	Currency           string          `json:"currency"`
	Year               int             `json:"year"`
	Inflow             decimal.Decimal `json:"inflow"`
	Outflow            decimal.Decimal `json:"outflow"`
	Net                decimal.Decimal `json:"net"`
	AvgMonthlyInflow   decimal.Decimal `json:"avg_monthly_inflow"`
	AvgMonthlyOutflow  decimal.Decimal `json:"avg_monthly_outflow"`
	TakeHome           decimal.Decimal `json:"take_home"`
	Overflow           decimal.Decimal `json:"overflow"`
	AvgMonthlyTakeHome decimal.Decimal `json:"avg_monthly_take_home"`
	AvgMonthlyOverflow decimal.Decimal `json:"avg_monthly_overflow"`
	ActiveMonths       int             `json:"active_months"`
	Categories         []CategoryStat  `json:"categories,omitempty"`
	GeneratedAt        time.Time       `json:"generated_at"`
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

type YearlyTotalsRow struct {
	Year         int
	InflowText   string
	OutflowText  string
	NetText      string
	ActiveMonths int
}

type YearlyCategoryRow struct {
	Year        int
	CategoryID  int64
	DisplayName *string
	InflowText  string
	OutflowText string
	NetText     string
}

type MonthlyTotalsRow struct {
	Month       int
	InflowText  string
	OutflowText string
	NetText     string
}

type CurrentMonthStats struct {
	UserID            int64           `json:"user_id"`
	AccountID         *int64          `json:"account_id,omitempty"`
	Currency          string          `json:"currency"`
	Year              int             `json:"year"`
	Month             int             `json:"month"`
	Inflow            decimal.Decimal `json:"inflow"`
	Outflow           decimal.Decimal `json:"outflow"`
	Net               decimal.Decimal `json:"net"`
	TakeHome          decimal.Decimal `json:"take_home"`
	Overflow          decimal.Decimal `json:"overflow"`
	Savings           decimal.Decimal `json:"savings"`
	Investments       decimal.Decimal `json:"investments"`
	DebtRepayments    decimal.Decimal `json:"debt_repayments"`
	SavingsRate       decimal.Decimal `json:"savings_rate"`
	InvestRate        decimal.Decimal `json:"investments_rate"`
	DebtRepaymentRate decimal.Decimal `json:"debt_repayment_rate"`
	GeneratedAt       time.Time       `json:"generated_at"`
	Categories        []CategoryStat  `json:"categories,omitempty"`
}

type TodayStats struct {
	UserID      int64           `json:"user_id"`
	AccountID   *int64          `json:"account_id,omitempty"`
	Currency    string          `json:"currency"`
	Year        int             `json:"year"`
	Inflow      decimal.Decimal `json:"inflow"`
	Outflow     decimal.Decimal `json:"outflow"`
	Net         decimal.Decimal `json:"net"`
	GeneratedAt time.Time       `json:"generated_at"`
}
