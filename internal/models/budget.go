package models

import "time"

type MonthlyBudget struct {
	ID                uint                      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrganizationID    uint                      `gorm:"not_null;index" json:"organization_id"`
	UserID            uint                      `gorm:"not null" json:"user_id"`
	DynamicCategoryID uint                      `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"dynamic_category_id"`
	DynamicCategory   DynamicCategory           `gorm:"foreignKey:DynamicCategoryID" json:"dynamic_category"`
	Month             int                       `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"month"`
	Year              int                       `gorm:"not null;uniqueIndex:unique_org_dyn_year_month" json:"year"`
	TotalInflow       float64                   `gorm:"type:decimal(15,2);not null" json:"total_inflow"`
	TotalOutflow      float64                   `gorm:"type:decimal(15,2);not null" json:"total_outflow"`
	BudgetInflow      float64                   `gorm:"type:decimal(15,2);not null" json:"budget_inflow"`
	BudgetOutflow     float64                   `gorm:"type:decimal(15,2);not null" json:"budget_outflow"`
	EffectiveBudget   float64                   `gorm:"type:decimal(15,2);not null" json:"effective_budget"`
	BudgetSnapshot    float64                   `gorm:"type:decimal(15,2);not null" json:"budget_snapshot"`
	SnapshotThreshold float64                   `gorm:"type:decimal(15,2);not null" json:"snapshot_threshold"`
	Allocations       []MonthlyBudgetAllocation `json:"allocations"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type MonthlyBudgetUpdate struct {
	ID                uint     `gorm:"primaryKey;autoIncrement" json:"id"`
	BudgetSnapshot    *float64 `gorm:"type:decimal(15,2);not null" json:"budget_snapshot"`
	SnapshotThreshold *float64 `gorm:"type:decimal(15,2);not null" json:"snapshot_threshold"`
}

type MonthlyBudgetAllocation struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	MonthlyBudgetID uint      `gorm:"not null;uniqueIndex:unique_mb_category" json:"monthly_budget_id"`
	Category        string    `gorm:"type:enum('savings','investments','other');not null;uniqueIndex:unique_mb_category" json:"category"`
	Method          string    `gorm:"type:enum('percentile', 'absolute');not null" json:"method"`
	Allocation      float64   `gorm:"type:decimal(15,2);not null" json:"allocation"`
	AllocatedValue  float64   `gorm:"type:decimal(15,2);not null" json:"allocated_value"`
	UsedValue       *float64  `gorm:"type:decimal(15,2)" json:"used_value"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (MonthlyBudget) TableName() string {
	return "monthly_budget"
}
