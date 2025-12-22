package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvestmentType string

const (
	InvestmentStock  InvestmentType = "stock"
	InvestmentETF    InvestmentType = "etf"
	InvestmentCrypto InvestmentType = "crypto"
)

type InvestmentHolding struct {
	ID              int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID       int64            `gorm:"not null;index:idx_holdings_account" json:"account_id"`
	UserID          int64            `gorm:"not null;index:idx_holdings_user" json:"user_id"`
	InvestmentType  InvestmentType   `gorm:"type:investment_type;not null" json:"investment_type"`
	Name            string           `gorm:"type:varchar(255);not null" json:"name"`
	Ticker          string           `gorm:"type:varchar(20);not null;index:idx_holdings_ticker" json:"ticker"`
	Quantity        decimal.Decimal  `gorm:"type:decimal(19,8);not null" json:"quantity"`
	AverageBuyPrice decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"average_buy_price"`
	CurrentPrice    *decimal.Decimal `gorm:"type:decimal(19,4)" json:"current_price"`
	LastPriceUpdate *time.Time       `json:"last_price_update"`
	Account         Account          `json:"account"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

type TransactionType string

const (
	InvestmentBuy  TransactionType = "buy"
	InvestmentSell TransactionType = "sell"
)

type InvestmentTransaction struct {
	ID                int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int64             `gorm:"not null" json:"user_id"`
	HoldingID         int64             `gorm:"not null;index:idx_inv_trans_holding" json:"holding_id"`
	TxnDate           time.Time         `gorm:"type:date;not null;index:idx_inv_trans_date" json:"txn_date"`
	TransactionType   TransactionType   `gorm:"type:varchar(4);not null" json:"transaction_type"`
	Quantity          decimal.Decimal   `gorm:"type:decimal(19,8);not null" json:"quantity"`
	Fee               decimal.Decimal   `gorm:"type:decimal(19,4);not null;default:0" json:"fee"`
	PricePerUnit      decimal.Decimal   `gorm:"type:decimal(19,4);not null" json:"price_per_unit"`
	ValueAtBuy        decimal.Decimal   `gorm:"type:decimal(19,4);not null" json:"value_at_buy"`
	Currency          string            `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	ExchangeRateToUSD decimal.Decimal   `gorm:"type:decimal(19,6);not null;default:1.0" json:"exchange_rate_to_usd"`
	Description       *string           `gorm:"type:varchar(255)" json:"description"`
	Holding           InvestmentHolding `json:"holding"`
	CreatedAt         time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

type InvestmentHoldingReq struct {
	AccountID      int64           `json:"account_id" validate:"required"`
	InvestmentType InvestmentType  `json:"investment_type" validate:"required"`
	Name           string          `json:"name" validate:"required"`
	Ticker         string          `json:"ticker" validate:"required"`
	Quantity       decimal.Decimal `json:"quantity" validate:"required"`
}

type InvestmentTransactionReq struct {
	HoldingID       int64            `json:"holding_id" validate:"required"`
	TransactionType TransactionType  `json:"transaction_type" validate:"required"`
	TxnDate         time.Time        `json:"txn_date" validate:"required"`
	Quantity        decimal.Decimal  `json:"quantity" validate:"required"`
	PricePerUnit    decimal.Decimal  `json:"price_per_unit" validate:"required"`
	Currency        string           `json:"currency" validate:"required"`
	Fee             *decimal.Decimal `json:"fee"`
	Description     *string          `json:"description,omitempty"`
}
