package models

import "time"

type OutflowCategory struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"not_null;index" json:"organization_id"`
	Name           string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_outflow_category_name" json:"name"`
	OutflowType    string    `gorm:"type:varchar(100);not null;" json:"outflow_type"`
	SpendingLimit  float64   `gorm:"type:decimal(10,2);" json:"spending_limit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Outflow struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	OrganizationID    uint            `gorm:"not_null" json:"organization_id"`
	OutflowCategoryID uint            `gorm:"index" json:"outflow_category_id"`
	OutflowCategory   OutflowCategory `gorm:"foreignKey:OutflowCategoryID" json:"outflow_category"`
	Amount            float64         `gorm:"type:decimal(10,2);not null;check:amount >= 0 AND amount <= 1000000000" json:"amount"`
	Description       *string         `gorm:"" json:"description"`
	OutflowDate       time.Time       `gorm:"not null" json:"outflow_date"`
	DeletedAt         *time.Time      `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type OutflowSummary struct {
	Month         int      `json:"month"`
	CategoryID    uint     `json:"category_id" gorm:"column:category_id"`
	CategoryName  string   `json:"category_name" gorm:"column:category_name"`
	CategoryType  string   `json:"category_type" gorm:"column:category_type"`
	TotalAmount   float64  `json:"total_amount" gorm:"column:total_amount"`
	SpendingLimit *float64 `json:"spending_limit"`
}
