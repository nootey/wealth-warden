package models

import "time"

type Transaction struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint      `gorm:"not null;index:idx_transactions_user_date" json:"user_id"`
	AccountID       uint      `gorm:"not null;index:idx_transactions_account_date" json:"account_id" validate:"required"`
	CategoryID      *uint     `gorm:"index:idx_transactions_category" json:"category_id,omitempty"`
	TransactionType string    `gorm:"not null;" json:"transaction_type" validate:"required"`
	Amount          float64   `gorm:"type:decimal(19,4);not null" json:"amount"  validate:"required"`
	Currency        string    `gorm:"type:char(3);not null;default:'EUR'" json:"currency" validate:"required"`
	TxnDate         time.Time `gorm:"type:date;not null;index" json:"txn_date" validate:"required"`
	Description     *string   `gorm:"type:varchar(255)" json:"description,omitempty"`
	Account         Account   `json:"account"`
	Category        Category  `json:"category,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Category struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         *uint     `gorm:"index:idx_categories_user_class" json:"user_id"`
	Name           string    `gorm:"type:varchar(100);not null" json:"name"`
	Classification string    `gorm:"type:enum('income','expense','savings','investment');not null;default:'expense';index:idx_categories_user_class" json:"classification"`
	ParentID       *uint     `gorm:"index" json:"parent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TransactionCreateReq struct {
	AccountID       uint      `json:"account_id" validate:"required"`
	CategoryID      *uint     `json:"category_id,omitempty"`
	TransactionType string    `json:"transaction_type" validate:"required"`
	Amount          float64   `json:"amount" validate:"required"`
	TxnDate         time.Time `json:"txn_date" validate:"required"`
	Description     *string   `json:"description,omitempty"`
}
