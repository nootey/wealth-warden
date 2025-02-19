package models

import "time"

type InflowCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not_null;index" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Inflow struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"not_null" json:"user_id"`
	InflowCategoryID uint           `gorm:"index" json:"inflow_category_id"`
	InflowCategory   InflowCategory `gorm:"foreignKey:InflowCategoryID" json:"inflow_category"`
	Amount           float64        `gorm:"type:decimal(10,2);not null;check:amount >= 0 AND amount <= 1000000000" json:"amount"`
	InflowDate       time.Time      `gorm:"not null" json:"inflow_date"`
	DeletedAt        *time.Time     `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type InflowSummary struct {
	Month              int     `json:"month"`
	TotalAmount        float64 `json:"total_amount"`
	InflowCategoryID   uint    `json:"inflow_category_id"`
	InflowCategoryName string  `json:"inflow_category_name"`
}
