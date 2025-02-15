package models

import "time"

type InflowCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Inflow struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	InflowCategoryID uint            `gorm:"index" json:"inflow_category_id"`
	InflowCategory   *InflowCategory `gorm:"foreignKey:InflowCategoryID" json:"inflow_category,omitempty"`
	Amount           float64         `gorm:"type:decimal(10,2);not null" json:"amount"`
	InflowDate       time.Time       `gorm:"not null" json:"inflow_date"`
	DeletedAt        *time.Time      `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type RecurringInflow struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	InflowCategoryID uint            `gorm:"index" json:"inflow_category_id"`
	InflowCategory   *InflowCategory `gorm:"foreignKey:InflowCategoryID" json:"inflow_category,omitempty"`
	Amount           float64         `gorm:"type:decimal(10,2);not null" json:"amount"`
	StartDate        time.Time       `gorm:"type:date;not null" json:"start_date"`
	EndDate          *time.Time      `gorm:"type:date;default:null" json:"end_date,omitempty"`
	Frequency        string          `gorm:"type:enum('daily', 'weekly', 'monthly', 'yearly');not null" json:"frequency"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `gorm:"index" json:"deleted_at,omitempty"`
}
