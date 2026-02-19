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

type YearlyCashflowBreakdown struct {
	Year   int              `json:"year"`
	Months []MonthBreakdown `json:"months"`
}

type MonthBreakdown struct {
	Month      int             `json:"month"`
	Categories MonthCategories `json:"categories"`
}

type MonthCategories struct {
	Inflows        decimal.Decimal `json:"inflows"`
	Outflows       decimal.Decimal `json:"outflows"`
	Investments    decimal.Decimal `json:"investments"`
	Savings        decimal.Decimal `json:"savings"`
	DebtRepayments decimal.Decimal `json:"debt_repayments"`
	TakeHome       decimal.Decimal `json:"take_home"`
	Overflow       decimal.Decimal `json:"overflow"`
}

type YearlySankeyData struct {
	Year     int    `json:"year"`
	Currency string `json:"currency"`

	// Starting point
	TotalIncome decimal.Decimal `json:"total_income"`

	// Primary allocations (first level flows)
	Savings        decimal.Decimal `json:"savings"`
	Investments    decimal.Decimal `json:"investments"`
	DebtRepayments decimal.Decimal `json:"debt_repayments"`
	Expenses       decimal.Decimal `json:"expenses"`

	// Second level flows - expense breakdown by category
	ExpenseCategories []CategoryFlow `json:"expense_categories,omitempty"`
}

type CategoryFlow struct {
	CategoryID   int64           `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Amount       decimal.Decimal `json:"amount"`
	Percentage   decimal.Decimal `json:"percentage"`
}

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

type YearlyBreakdownStats struct {
	CurrentYear    *YearStatsWithAllocations `json:"current_year"`
	ComparisonYear *YearStatsWithAllocations `json:"comparison_year,omitempty"`
}

type YearStatsWithAllocations struct {
	Year int `json:"year"`

	Inflow            decimal.Decimal `json:"inflow"`
	Outflow           decimal.Decimal `json:"outflow"`
	AvgMonthlyInflow  decimal.Decimal `json:"avg_monthly_inflow"`
	AvgMonthlyOutflow decimal.Decimal `json:"avg_monthly_outflow"`

	TakeHome           decimal.Decimal `json:"take_home"`
	Overflow           decimal.Decimal `json:"overflow"`
	AvgMonthlyTakeHome decimal.Decimal `json:"avg_monthly_take_home"`
	AvgMonthlyOverflow decimal.Decimal `json:"avg_monthly_overflow"`

	SavingsAllocated    decimal.Decimal `json:"savings_allocated"`
	InvestmentAllocated decimal.Decimal `json:"investment_allocated"`
	DebtAllocated       decimal.Decimal `json:"debt_allocated"`
	TotalAllocated      decimal.Decimal `json:"total_allocated"`

	SavingsPct    float64 `json:"savings_pct"`
	InvestmentPct float64 `json:"investment_pct"`
	DebtPct       float64 `json:"debt_pct"`
}
