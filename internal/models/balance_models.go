package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	AccountID int64           `gorm:"primaryKey" json:"account_id"`
	UserID    int64           `gorm:"not null" json:"user_id"`
	Currency  string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	Balance   decimal.Decimal `gorm:"column:balance;type:numeric(19,4);not null;default:0" json:"balance"`
	CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// AccountBalance is the read view: cash plus the latest snapshot's market value.
// It is never written
type AccountBalance struct {
	AccountID    int64           `json:"account_id"`
	Currency     string          `json:"currency"`
	Balance      decimal.Decimal `json:"balance"`
	MarketValue  decimal.Decimal `json:"market_value"`
	TotalBalance decimal.Decimal `json:"total_balance"`
}

type AccountDailySnapshot struct {
	ID          int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64           `gorm:"not null" json:"user_id"`
	AccountID   int64           `gorm:"not null" json:"account_id"`
	AsOf        time.Time       `gorm:"type:date;not null;" json:"as_of"`
	EndBalance  decimal.Decimal `gorm:"type:numeric(19,4);not null" json:"end_balance"`
	MarketValue decimal.Decimal `gorm:"type:numeric(19,4);not null;default:0" json:"market_value"`
	Currency    string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	ComputedAt  time.Time       `gorm:"autoCreateTime" json:"computed_at"`
}
