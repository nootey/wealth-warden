package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type Transaction struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          int64           `gorm:"not null;index:idx_transactions_user_date" json:"user_id"`
	AccountID       int64           `gorm:"not null;index:idx_transactions_account_date" json:"account_id"`
	CategoryID      *int64          `gorm:"index:idx_transactions_category" json:"category_id,omitempty"`
	TransactionType string          `gorm:"not null;" json:"transaction_type"`
	Amount          decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Currency        string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	TxnDate         time.Time       `gorm:"not null;index" json:"txn_date"`
	Description     *string         `gorm:"type:varchar(255)" json:"description,omitempty"`
	Account         Account         `json:"account"`
	Category        Category        `json:"category,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Transfer struct {
	ID                   int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionInflowID  int64           `gorm:"not null;index:idx_transfer_transaction_inflow" json:"transaction_inflow_id"`
	TransactionOutflowID int64           `gorm:"not null;index:idx_transfer_transaction_outflow" json:"transaction_outflow_id"`
	Amount               decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Currency             string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Status               string          `gorm:"not null" json:"status"`
	Notes                *string         `gorm:"type:text" json:"notes"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type Category struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         *int64    `gorm:"index:idx_categories_user_class" json:"user_id"`
	Name           string    `gorm:"type:varchar(100);not null" json:"name"`
	Classification string    `gorm:"type:enum('income','expense','savings','investment');not null;default:'expense';index:idx_categories_user_class" json:"classification"`
	ParentID       *int64    `gorm:"index" json:"parent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type HiddenCategory struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not_null" json:"user_id"`
	CategoryID int64     `gorm:"not_null" json:"category_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TransactionReq struct {
	AccountID       int64           `json:"account_id" validate:"required"`
	CategoryID      *int64          `json:"category_id,omitempty"`
	TransactionType string          `json:"transaction_type" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	TxnDate         time.Time       `json:"txn_date" validate:"required"`
	Description     *string         `json:"description,omitempty"`
}

type TransferReq struct {
	TransactionInflowID  int64           `json:"transaction_inflow_id" validate:"required"`
	TransactionOutflowID *int64          `json:"transaction_outflow_id" validate:"required"`
	Amount               decimal.Decimal `json:"amount" validate:"required"`
	Notes                *string         `json:"notes"`
}
