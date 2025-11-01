package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          int64           `gorm:"not null;index:idx_transactions_user_date" json:"user_id"`
	AccountID       int64           `gorm:"not null;index:idx_transactions_account_date" json:"account_id"`
	CategoryID      *int64          `gorm:"index:idx_transactions_category" json:"category_id,omitempty"`
	ImportID        *int64          `json:"import_id,omitempty"`
	TransactionType string          `gorm:"not null;enum(income,expense)" json:"transaction_type"`
	Amount          decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Currency        string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	TxnDate         time.Time       `gorm:"not null;index" json:"txn_date"`
	Description     *string         `gorm:"type:varchar(255)" json:"description,omitempty"`
	IsAdjustment    bool            `gorm:"not null;type:boolean" json:"is_adjustment"`
	IsTransfer      bool            `gorm:"not null;type:boolean" json:"is_transfer"`
	Account         Account         `json:"account"`
	Category        Category        `json:"category,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

type Transfer struct {
	ID                   int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID               int64           `gorm:"not null" json:"user_id"`
	TransactionInflowID  int64           `gorm:"not null;index:idx_transfer_transaction_inflow" json:"transaction_inflow_id"`
	TransactionOutflowID int64           `gorm:"not null;index:idx_transfer_transaction_outflow" json:"transaction_outflow_id"`
	ImportID             *int64          `json:"import_id,omitempty"`
	Amount               decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Currency             string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Status               string          `gorm:"not null" json:"status"`
	Notes                *string         `gorm:"type:text" json:"notes"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            *time.Time      `json:"deleted_at"`

	// Relations
	TransactionInflow  Transaction `gorm:"foreignKey:TransactionInflowID;references:ID" json:"to"`
	TransactionOutflow Transaction `gorm:"foreignKey:TransactionOutflowID;references:ID" json:"from"`
}

type TransactionTemplate struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string          `gorm:"varchar(150)" json:"name"`
	UserID          int64           `gorm:"not null" json:"user_id"`
	AccountID       int64           `gorm:"not null" json:"account_id"`
	CategoryID      int64           `gorm:"not null;" json:"category_id,omitempty"`
	TransactionType string          `gorm:"not null;enum(income,expense)" json:"transaction_type"`
	Amount          decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Frequency       string          `gorm:"not null:enum(weekly,biweekly,monthly,quarterly,annually)" json:"frequency"`
	NextRunAt       time.Time       `gorm:"not null" json:"next_run_at"`
	LastRunAt       *time.Time      `json:"last_run_at"`
	RunCount        int             `gorm:"not null;default:0" json:"run_count"`
	EndDate         *time.Time      `json:"end_date"`
	MaxRuns         *int            `json:"max_runs"`
	IsActive        bool            `gorm:"not null;default:true" json:"is_active"`
	Account         Account         `json:"account"`
	Category        Category        `json:"category,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Category struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         *int64     `gorm:"index:idx_categories_user_class" json:"user_id"`
	Name           string     `gorm:"type:varchar(100);not null" json:"name"`
	DisplayName    string     `gorm:"type:varchar(100);not null" json:"display_name"`
	Classification string     `gorm:"not null;default:'expense';index:idx_categories_user_class" json:"classification"`
	ParentID       *int64     `gorm:"index" json:"parent_id,omitempty"`
	IsDefault      bool       `json:"is_default"`
	ImportID       *int64     `json:"import_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
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
	SourceID      int64           `json:"source_id" validate:"required"`
	DestinationID int64           `json:"destination_id" validate:"required"`
	Amount        decimal.Decimal `json:"amount" validate:"required"`
	Notes         *string         `json:"notes"`
	CreatedAt     time.Time       `json:"created_at"`
}

type TrRestoreReq struct {
	ID int64 `json:"id" validate:"required"`
}

type CategoryReq struct {
	DisplayName    string `json:"display_name" validate:"required"`
	Classification string `json:"classification" validate:"required"`
}

type TransactionTemplateReq struct {
	Name            string          `json:"name" validate:"required"`
	AccountID       int64           `json:"account_id" validate:"required"`
	CategoryID      int64           `json:"category_id" validate:"required"`
	TransactionType string          `json:"transaction_type" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	Frequency       string          `gorm:"not null:enum(weekly,biweekly,monthly,quarterly,annually)" json:"frequency" validate:"required"`
	NextRunAt       time.Time       `gorm:"not null" json:"next_run_at" validate:"required"`
	EndDate         *time.Time      `json:"end_date"`
	MaxRuns         *int            `json:"max_runs"`
	IsActive        bool            `json:"is_active" validate:"required"`
}
