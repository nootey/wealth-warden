package models

import (
	"time"
)

type Import struct {
	ID                     int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                   string     `gorm:"size:128;not null" json:"name"`
	UserID                 int64      `gorm:"not null" json:"user_id"`
	Type                   string     `gorm:"not null" json:"type"`
	SubType                string     `gorm:"not null" json:"sub_type"`
	Status                 string     `gorm:"not null" json:"status"`
	Currency               string     `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Step                   string     `json:"step"`
	InvestmentsTransferred bool       `json:"investments_transferred"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	StartedAt              *time.Time `json:"started_at"`
	CompletedAt            *time.Time `json:"completed_at"`
}

type AccImportPayload struct {
	GeneratedAt time.Time       `json:"generated_at" validate:"required"`
	Accounts    []AccountExport `json:"accounts" validate:"required"`
}

type TxnImportPayload struct {
	GeneratedAt      time.Time         `json:"generated_at" validate:"required"`
	Txns             []JSONTxn         `json:"transactions" validate:"required"`
	Transfers        []JSONTxn         `json:"transfers,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	CategoryMappings []CategoryMapping `json:"category_mappings" validate:"required"`
}

type InvestmentTransferPayload struct {
	ImportID           int64               `json:"import_id" validate:"required"`
	CheckingAccID      int64               `json:"checking_acc_id" validate:"required"`
	InvestmentMappings []InvestmentMapping `json:"investment_mappings" validate:"required"`
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
