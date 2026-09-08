package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	AccountID    int64           `gorm:"primaryKey" json:"account_id"`
	UserID       int64           `gorm:"not null" json:"user_id"`
	Currency     string          `gorm:"type:char(3);not null;default:'EUR'" json:"currency"`
	EndBalance   decimal.Decimal `gorm:"column:balance;type:numeric(19,4);not null;default:0" json:"end_balance"`
	StartBalance decimal.Decimal `gorm:"-" json:"start_balance"` // opening amount, summed from transactions
	MarketValue  decimal.Decimal `gorm:"-" json:"market_value"`  // latest snapshot, for investment and crypto accounts
	TotalBalance decimal.Decimal `gorm:"-" json:"total_balance"` // end_balance + market_value
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}
