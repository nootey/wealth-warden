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
	Description      *string        `gorm:"" json:"description"`
	InflowDate       time.Time      `gorm:"not null" json:"inflow_date"`
	DeletedAt        *time.Time     `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type InflowSummary struct {
	Month        int     `json:"month"`
	CategoryID   uint    `json:"category_id" gorm:"column:category_id"`
	CategoryName string  `json:"category_name" gorm:"column:category_name"`
	TotalAmount  float64 `json:"total_amount" gorm:"column:total_amount"`
}

type DynamicCategory struct {
	ID        uint                     `gorm:"primaryKey" json:"id"`
	UserID    uint                     `gorm:"not_null;index" json:"user_id"`
	Name      string                   `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_name" json:"name"`
	Mappings  []DynamicCategoryMapping `gorm:"foreignKey:DynamicCategoryID"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type DynamicCategoryMapping struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	DynamicCategoryID   uint      `gorm:"column:dynamic_category_id;not_null;index" json:"dynamic_category_id"`
	RelatedCategoryID   uint      `gorm:"column:related_id;not_null;index" json:"related_id"`
	RelatedCategoryName string    `gorm:"column:related_type;not_null" json:"related_type"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
