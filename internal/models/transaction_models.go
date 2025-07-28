package models

import "time"

type Transaction struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64    `gorm:"not null;index:idx_transactions_user_date" json:"user_id"`
	AccountID       uint64    `gorm:"not null;index:idx_transactions_account_date" json:"account_id"`
	CategoryID      *uint64   `gorm:"index:idx_transactions_category" json:"category_id,omitempty"`
	TransactionType string    `gorm:"type:enum('increase','decrease','adjustment','transfer');not null;default:'decrease'" json:"transaction_type"`
	Amount          float64   `gorm:"type:decimal(19,4);not null" json:"amount"`
	Currency        string    `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	TxnDate         time.Time `gorm:"type:date;not null;index:idx_transactions_user_date,idx_transactions_account_date" json:"txn_date"`
	Description     *string   `gorm:"type:varchar(255)" json:"description,omitempty"`
	ReferenceID     *uint64   `gorm:"index:idx_transactions_reference" json:"reference_id,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
