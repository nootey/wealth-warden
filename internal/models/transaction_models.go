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
	IdempotencyKey  *string         `gorm:"type:varchar(64)" json:"idempotency_key,omitempty"`
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
	IdempotencyKey       *string         `gorm:"type:varchar(64)" json:"idempotency_key,omitempty"`
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
	ToAccountID     *int64          `json:"to_account_id,omitempty"`
	CategoryID      *int64          `json:"category_id,omitempty"`
	TemplateType    string          `gorm:"not null;default:'transaction'" json:"template_type"`
	TransactionType *string         `gorm:"enum(income,expense)" json:"transaction_type,omitempty"`
	Amount          decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	Frequency       string          `gorm:"not null:enum(weekly,biweekly,monthly,quarterly,annually)" json:"frequency"`
	DayOfMonth      int             `gorm:"not null;default:0" json:"day_of_month"`
	NextRunAt       time.Time       `gorm:"not null" json:"next_run_at"`
	LastRunAt       *time.Time      `json:"last_run_at"`
	RunCount        int             `gorm:"not null;default:0" json:"run_count"`
	EndDate         *time.Time      `json:"end_date"`
	MaxRuns         *int            `json:"max_runs"`
	IsActive        bool            `gorm:"not null;default:true" json:"is_active"`
	Account         Account         `json:"account"`
	ToAccount       *Account        `gorm:"foreignKey:ToAccountID" json:"to_account,omitempty"`
	Category        *Category       `json:"category,omitempty"`
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

type CategoryGroup struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         *int64    `gorm:"index:idx_categories_user_class" json:"user_id"`
	Name           string    `gorm:"type:varchar(100);not null" json:"name"`
	Classification string    `gorm:"not null" json:"classification"`
	Description    *string   `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Categories []Category `gorm:"many2many:category_group_members;joinForeignKey:group_id;joinReferences:category_id" json:"categories"`
}

type CategoryOrGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	IsGroup        bool    `json:"is_group"`
	Classification string  `json:"classification"`
	CategoryIDs    []int64 `json:"category_ids"`
}

type InsertResult struct {
	ID          int64
	IsDuplicate bool
}

type TransactionBatchTotals struct {
	Count    int64           `json:"count"`
	Income   decimal.Decimal `json:"income"`
	Expenses decimal.Decimal `json:"expenses"`
}

type TemplateSummary struct {
	MonthlyExpense    decimal.Decimal `json:"monthly_expense"`
	MonthlyIncome     decimal.Decimal `json:"monthly_income"`
	ThisMonthExpense  decimal.Decimal `json:"this_month_expense"`
	ThisMonthIncome   decimal.Decimal `json:"this_month_income"`
	MonthlyTransfer   decimal.Decimal `json:"monthly_transfer"`
	ThisMonthTransfer decimal.Decimal `json:"this_month_transfer"`
}

type TransactionReq struct {
	AccountID       int64           `json:"account_id" validate:"required"`
	CategoryID      *int64          `json:"category_id,omitempty"`
	TransactionType string          `json:"transaction_type" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	TxnDate         time.Time       `json:"txn_date" validate:"required"`
	Description     *string         `json:"description,omitempty"`
	IdempotencyKey  *string         `json:"idempotency_key,omitempty"`
}

type TransferReq struct {
	SourceID       int64           `json:"source_id" validate:"required"`
	DestinationID  int64           `json:"destination_id" validate:"required"`
	Amount         decimal.Decimal `json:"amount" validate:"required"`
	Notes          *string         `json:"notes"`
	CreatedAt      time.Time       `json:"created_at"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
}

type TrRestoreReq struct {
	ID int64 `json:"id" validate:"required"`
}

type CategoryReq struct {
	DisplayName    string `json:"display_name" validate:"required"`
	Classification string `json:"classification" validate:"required"`
}

type CategoryGroupReq struct {
	Name               string      `json:"name" validate:"required"`
	Classification     string      `json:"classification" validate:"required"`
	Description        *string     `json:"description"`
	SelectedCategories interface{} `json:"selected_categories" validate:"required"`
}

type UpdateTransferReq struct {
	Amount    decimal.Decimal `json:"amount" validate:"required"`
	Notes     *string         `json:"notes"`
	CreatedAt time.Time       `json:"created_at" validate:"required"`
}

type TransactionTemplateReq struct {
	Name            string          `json:"name" validate:"required"`
	TemplateType    string          `json:"template_type" validate:"required"`
	AccountID       int64           `json:"account_id" validate:"required"`
	ToAccountID     *int64          `json:"to_account_id,omitempty"`
	CategoryID      *int64          `json:"category_id,omitempty"`
	TransactionType *string         `json:"transaction_type,omitempty"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	Frequency       string          `json:"frequency" validate:"required"`
	NextRunAt       time.Time       `json:"next_run_at" validate:"required"`
	EndDate         *time.Time      `json:"end_date"`
	MaxRuns         *int            `json:"max_runs"`
	IsActive        bool            `json:"is_active" validate:"required"`
}
