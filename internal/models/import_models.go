package models

import (
	"time"
)

type Import struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	UserID      int64      `gorm:"not null" json:"user_id"`
	AccountID   int64      `gorm:"not null" json:"account_id"`
	ImportType  string     `gorm:"not null" json:"import_type"`
	Status      string     `gorm:"not null" json:"status"`
	Currency    string     `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Step        string     `json:"step"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type CustomImportPayload struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	Txns             []JSONTxn         `json:"transactions"`
	Transfers        []JSONTxn         `json:"transfers,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	CategoryMappings []CategoryMapping `json:"category_mappings"`
}

type JSONTxn struct {
	TransactionType string    `json:"transaction_type"`
	Amount          string    `json:"amount"`
	Currency        string    `json:"currency"`
	TxnDate         time.Time `json:"txn_date"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
}

type CategoryMapping struct {
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id"`
}

type InvestmentMapping struct {
	Name      string `json:"name"`
	AccountID int64  `json:"account_id"`
}
