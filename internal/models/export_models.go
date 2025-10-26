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

type AccountExport struct {
	Name        string `json:"name"`
	AccountType struct {
		Type           string `json:"type"`
		SubType        string `json:"sub_type"`
		Classification string `json:"classification"`
	} `json:"account_type"`
	Balance struct {
		StartBalance    string `json:"start_balance"`
		CashInflows     string `json:"cash_inflows"`
		CashOutflows    string `json:"cash_outflows"`
		NonCashInflows  string `json:"non_cash_inflows"`
		NonCashOutflows string `json:"non_cash_outflows"`
		NetMarketFlows  string `json:"net_market_flows"`
		Adjustments     string `json:"adjustments"`
	} `json:"balance"`
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
