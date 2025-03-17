package validators

import (
	"time"
)

type CreateInflowRequest struct {
	ID               uint      `json:"id"`
	InflowCategoryID uint      `json:"inflow_category_id" validate:"required,numeric"`
	Amount           float64   `json:"amount" validate:"required,numeric,min=0,max=1000000000"`
	InflowDate       time.Time `json:"inflow_date" validate:"required"`
	Description      string    `json:"description"`
}

type CreateInflowCategoryRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name" validate:"required,max=100"`
}

type CreateReoccurringActionRequest struct {
	StartDate     time.Time  `json:"start_date" validate:"required"`
	EndDate       *time.Time `json:"end_date"`
	Category      string     `json:"category_type" validate:"required"`
	IntervalUnit  string     `json:"interval_unit" validate:"required"`
	IntervalValue int        `json:"interval_value" validate:"required"`
}

type ReoccurringInflowRequest struct {
	Inflow    CreateInflowRequest            `json:"Inflow" validate:"required"`
	RecInflow CreateReoccurringActionRequest `json:"RecInflow" validate:"required"`
}

type CreateOutflowRequest struct {
	ID                uint      `json:"id"`
	OutflowCategoryID uint      `json:"outflow_category_id" validate:"required,numeric"`
	Amount            float64   `json:"amount" validate:"required,numeric,min=0,max=1000000000"`
	OutflowDate       time.Time `json:"outflow_date" validate:"required"`
	Description       string    `json:"description"`
}

type CreateOutflowCategoryRequest struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name" validate:"required,max=100"`
	OutflowType   string  `json:"outflow_type" validate:"required"`
	SpendingLimit float64 `json:"spending_limit"`
}

type ReoccurringOutflowRequest struct {
	Outflow    CreateOutflowRequest           `json:"Outflow" validate:"required"`
	RecOutflow CreateReoccurringActionRequest `json:"RecOutflow" validate:"required"`
}

type CreateDynamicCategoryRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name" validate:"required"`
}

type Link struct {
	ID           uint   `json:"id" validate:"required"`
	Name         string `json:"name" validate:"required"`
	UserID       uint   `json:"user_id"`
	CategoryType string `json:"category_type" validate:"required"`
}

type CreateDynamicCategoryMappingRequest struct {
	ID             uint   `json:"id"`
	PrimaryLinks   []Link `json:"primary_links" validate:"required"`
	PrimaryType    string `json:"primary_type" validate:"required"`
	SecondaryLinks []Link `json:"secondary_links"`
	SecondaryType  string `json:"secondary_type"`
}

type DynamicCategoryMapRequest struct {
	Category CreateDynamicCategoryRequest        `json:"Category" validate:"required"`
	Mapping  CreateDynamicCategoryMappingRequest `json:"Mapping" validate:"required"`
}

type CreateMonthlyBudgetRequest struct {
	ID                uint `json:"id"`
	DynamicCategoryID uint `json:"dynamic_category_id" validate:"required,numeric"`
}

type CreateMonthlyBudgetAllocationRequest struct {
	ID              uint    `json:"id"`
	MonthlyBudgetID uint    `json:"monthly_budget_id" validate:"required,numeric"`
	Allocation      float64 `json:"allocation" validate:"required"`
	Category        string  `json:"category" validate:"required"`
}

type UpdateMonthlyBudgetRequest struct {
	BudgetID uint        `json:"budget_id" validate:"required,numeric"`
	Field    string      `json:"field" validate:"required"`
	Value    interface{} `json:"value" validate:"required"`
}
