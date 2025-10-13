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
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type CustomImportPayload struct {
	Year        int          `json:"year"`
	GeneratedAt time.Time    `json:"generated_at"`
	Totals      ImportTotals `json:"totals"`
	Txns        []JSONTxn    `json:"transactions"`
}

type ImportTotals struct {
	Inflow      string  `json:"income"`
	Expense     string  `json:"expense"`
	Investments string  `json:"investments"`
	Savings     *string `json:"savings"`
}

type JSONTxn struct {
	TransactionType string    `json:"transaction_type"`
	Amount          string    `json:"amount"`
	Currency        string    `json:"currency"`
	TxnDate         time.Time `json:"txn_date"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
}
