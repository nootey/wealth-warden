package models

import "time"

type MonthlyBudget struct {
	ID                uint64  `gorm:"primaryKey;autoIncrement"`
	OrganizationID    uint    `gorm:"not_null;index" json:"organization_id"`
	UserID            uint    `gorm:"not null" json:"user_id"`
	DynamicCategoryID uint64  `gorm:"not null;uniqueIndex:unique_org_dyn_year_month"`
	Month             uint8   `gorm:"not null;uniqueIndex:unique_org_dyn_year_month"` // Note: SQL CHECK (month BETWEEN 1 AND 12) must be validated in code.
	Year              int     `gorm:"not null;uniqueIndex:unique_org_dyn_year_month"`
	TotalInflow       float64 `gorm:"type:decimal(15,2);not null"`
	TotalOutflow      float64 `gorm:"type:decimal(15,2);not null"`
	EffectiveBudget   float64 `gorm:"type:decimal(15,2);->;not null"` // computed column: total_inflow - total_outflow
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MonthlyBudgetAllocation struct {
	ID                  uint64  `gorm:"primaryKey;autoIncrement"`
	MonthlyBudgetID     uint64  `gorm:"not null;uniqueIndex:unique_mb_category"`
	Category            string  `gorm:"type:enum('savings','investments','other');not null;uniqueIndex:unique_mb_category"`
	TotalAllocatedValue float64 `gorm:"type:decimal(15,2);not null"` // Validate value >= 0 in your application if needed
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (MonthlyBudget) TableName() string {
	return "monthly_budget"
}
