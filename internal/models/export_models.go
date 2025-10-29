package models

import (
	"time"
)

type Export struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	UserID      int64      `gorm:"not null;index" json:"user_id"`
	ExportType  string     `gorm:"size:128;not null" json:"export_type"`
	Status      string     `gorm:"not null;default:'pending'" json:"status"`
	Currency    string     `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	FilePath    *string    `gorm:"type:text" json:"file_path,omitempty"`
	FileSize    *int64     `json:"file_size,omitempty"`
	Error       *string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type AccountExport struct {
	Name        string `json:"name"`
	AccountType struct {
		Type           string `json:"type"`
		SubType        string `json:"sub_type"`
		Classification string `json:"classification"`
	} `json:"account_type"`
	Balance  string    `json:"balance" json:"balance"`
	Currency string    `json:"currency"`
	OpenedAt time.Time `json:"opened_at"`
}

type CategoryExport struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Classification string `json:"classification"`
	ParentID       *int64 `json:"parent_id,omitempty"`
	IsDefault      bool   `json:"is_default"`
}
