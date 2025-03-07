package models

import "time"

type RecurringAction struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	OrganizationID uint       `gorm:"index" json:"organization_id"`
	CategoryType   string     `gorm:"not null" json:"category_type"`
	CategoryID     uint       `gorm:"index" json:"action_id"`
	Amount         float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	StartDate      time.Time  `gorm:"type:date;not null" json:"start_date"`
	EndDate        *time.Time `gorm:"type:date;default:null" json:"end_date,omitempty"`
	IntervalValue  int        `gorm:"not null" json:"interval_value"`
	IntervalUnit   string     `gorm:"type:enum('days', 'weeks', 'months', 'years');not null" json:"interval_unit"`
	LastProcessed  *time.Time `gorm:"type:date;default:null" json:"last_processed,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
