package models

import "time"

type MonthlyBudget struct {
	ID                uint                      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrganizationID    uint                      `gorm:"not_null;index" json:"organization_id"`
	UserID            uint                      `gorm:"not null" json:"user_id"`
	DynamicCategoryID uint                      `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"dynamic_category_id"`
	DynamicCategory   DynamicCategory           `gorm:"foreignKey:DynamicCategoryID" json:"dynamic_category"`
	Month             int                       `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"month"` // Note: SQL CHECK (month BETWEEN 1 AND 12) must be validated in code.
	Year              int                       `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"year"`
	TotalInflow       float64                   `gorm:"type:decimal(15,2);not null" json:"total_inflow"`
	TotalOutflow      float64                   `gorm:"type:decimal(15,2);not null" json:"total_outflow"`
	EffectiveBudget   float64                   `gorm:"type:decimal(15,2);->;not null" json:"effective_budget"` // computed column: total_inflow - total_outflow
	BudgetSnapshot    float64                   `gorm:"type:decimal(15,2);->;not null" json:"budget_snapshot"`  // snapshot of effective budget, updated manually instead of via category
	Allocations       []MonthlyBudgetAllocation `json:"allocations"`                                            // snapshot of effective budget, updated manually instead of via category
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type MonthlyBudgetAllocation struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	MonthlyBudgetID     uint      `gorm:"not null;uniqueIndex:unique_mb_category" json:"monthly_budget_id"`
	Category            string    `gorm:"type:enum('savings','investments','other');not null;uniqueIndex:unique_mb_category" json:"category"`
	TotalAllocatedValue float64   `gorm:"type:decimal(15,2);not null" json:"total_allocated_value"` // Validate value >= 0 in your application if needed
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (MonthlyBudget) TableName() string {
	return "monthly_budget"
}
