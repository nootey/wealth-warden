package models

import (
	"time"

	"github.com/shopspring/decimal"
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
	Currency  string       `json:"currency"`
	Points    []ChartPoint `json:"points"`
	Current   ChartPoint   `json:"current"`
	Change    *Change      `json:"change,omitempty"`
	AssetType *string      `json:"asset_type"`
}

type MonthlyCashflow struct {
	Month    int             `json:"month"`
	Inflows  decimal.Decimal `json:"inflows"`
	Outflows decimal.Decimal `json:"outflows"`
	Net      decimal.Decimal `json:"net"`
}

type MonthlyCashflowResponse struct {
	Year   int               `json:"year"`
	Series []MonthlyCashflow `json:"series"`
}

type MonthlyCategoryUsage struct {
	Month      int              `json:"month"`
	CategoryID int64            `json:"category_id"`
	Category   string           `json:"category"`
	Amount     decimal.Decimal  `json:"amount"`
	Percentage *decimal.Decimal `json:"percentage,omitempty"`
}

type CategoryUsageResponse struct {
	Year   int                    `json:"year"`
	Class  string                 `json:"classification"`
	Series []MonthlyCategoryUsage `json:"series"`
}

type MultiYearCategoryUsageResponse struct {
	Years  []int                         `json:"years"`
	Class  string                        `json:"classification"`
	ByYear map[int]CategoryUsageResponse `json:"by_year"`
	Stats  MultiYearYCategoryStats       `json:"stats"`
}

type MultiYearYCategoryStats struct {
	YearStats     map[int]YearStat `json:"year_stats"`
	AllTimeTotal  decimal.Decimal  `json:"all_time_total"`
	AllTimeAvg    decimal.Decimal  `json:"all_time_avg"`
	AllTimeMonths int              `json:"all_time_months"`
}

type YearStat struct {
	Total          decimal.Decimal `json:"total"`
	MonthlyAvg     decimal.Decimal `json:"monthly_avg"`
	MonthsWithData int             `json:"months_with_data"`
}
