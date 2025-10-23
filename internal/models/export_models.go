package models

import (
	"time"
)

type Export struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	UserID      int64      `gorm:"not null" json:"user_id"`
	AccountID   int64      `gorm:"not null" json:"account_id"`
	ExportType  string     `gorm:"not null" json:"export_type"`
	Status      string     `gorm:"not null" json:"status"`
	Currency    string     `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Step        string     `json:"step"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}
