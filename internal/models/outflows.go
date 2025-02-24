package models

import "time"

type OutflowCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not_null;index" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Outflow struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	UserID            uint            `gorm:"not_null" json:"user_id"`
	OutflowCategoryID uint            `gorm:"index" json:"outflow_category_id"`
	OutflowCategory   OutflowCategory `gorm:"foreignKey:OutflowCategoryID" json:"outflow_category"`
	Amount            float64         `gorm:"type:decimal(10,2);not null;check:amount >= 0 AND amount <= 1000000000" json:"amount"`
	Description       string          `gorm:"" json:"description"`
	OutflowDate       time.Time       `gorm:"not null" json:"outflow_date"`
	DeletedAt         *time.Time      `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type OutflowSummary struct {
	Month               int     `json:"month"`
	TotalAmount         float64 `json:"total_amount"`
	OutflowCategoryID   uint    `json:"outflow_category_id"`
	OutflowCategoryName string  `json:"outflow_category_name"`
}
