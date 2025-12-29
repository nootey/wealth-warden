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

type InvestmentAsset struct {
	ID                int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID         int64            `gorm:"not null;index:idx_assets_account" json:"account_id"`
	UserID            int64            `gorm:"not null;index:idx_assets_user" json:"user_id"`
	InvestmentType    InvestmentType   `gorm:"type:investment_type;not null" json:"investment_type"`
	Name              string           `gorm:"type:varchar(255);not null" json:"name"`
	Ticker            string           `gorm:"type:varchar(20);not null;index:idx_assets_ticker" json:"ticker"`
	Quantity          decimal.Decimal  `gorm:"type:decimal(19,8);not null" json:"quantity"`
	AverageBuyPrice   decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"average_buy_price"`
	ValueAtBuy        decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"value_at_buy"`
	CurrentValue      decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"current_value"`
	CurrentPrice      *decimal.Decimal `gorm:"type:decimal(19,4)" json:"current_price"`
	ProfitLoss        decimal.Decimal  `gorm:"type:decimal(19,4);not null;default:0" json:"profit_loss"`
	ProfitLossPercent decimal.Decimal  `gorm:"type:decimal(10,2);not null;default:0" json:"profit_loss_percent"`
	LastPriceUpdate   *time.Time       `json:"last_price_update"`
	Currency          string           `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	Account           Account          `json:"account"`
	ImportID          *int64           `json:"import_id,omitempty"`
	CreatedAt         time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

type TradeType string

const (
	InvestmentBuy  TradeType = "buy"
	InvestmentSell TradeType = "sell"
)

type InvestmentTrade struct {
	ID                int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int64           `gorm:"not null" json:"user_id"`
	AssetID           int64           `gorm:"not null;index:idx_inv_trans_asset" json:"asset_id"`
	TxnDate           time.Time       `gorm:"type:date;not null;index:idx_inv_trans_date" json:"txn_date"`
	TradeType         TradeType       `gorm:"type:varchar(4);not null" json:"trade_type"`
	Quantity          decimal.Decimal `gorm:"type:decimal(19,8);not null" json:"quantity"`
	Fee               decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"fee"`
	PricePerUnit      decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"price_per_unit"`
	ValueAtBuy        decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"value_at_buy"`
	CurrentValue      decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"current_value"`
	RealizedValue     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"realized_value"`
	ProfitLoss        decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"profit_loss"`
	ProfitLossPercent decimal.Decimal `gorm:"type:decimal(10,2);not null;default:0" json:"profit_loss_percent"`
	Currency          string          `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	ExchangeRateToUSD decimal.Decimal `gorm:"type:decimal(19,6);not null;default:1.0" json:"exchange_rate_to_usd"`
	Description       *string         `gorm:"type:varchar(255)" json:"description"`
	Asset             InvestmentAsset `json:"asset"`
	ImportID          *int64          `json:"import_id,omitempty"`
	CreatedAt         time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

type InvestmentAssetReq struct {
	AccountID      int64           `json:"account_id" validate:"required"`
	InvestmentType InvestmentType  `json:"investment_type" validate:"required"`
	Name           string          `json:"name" validate:"required"`
	Ticker         string          `json:"ticker" validate:"required"`
	Quantity       decimal.Decimal `json:"quantity" validate:"required"`
	Currency       string          `json:"currency" validate:"required"`
}

type InvestmentTradeReq struct {
	AssetID      int64            `json:"asset_id" validate:"required"`
	TradeType    TradeType        `json:"trade_type" validate:"required"`
	TxnDate      time.Time        `json:"txn_date" validate:"required"`
	Quantity     decimal.Decimal  `json:"quantity" validate:"required"`
	PricePerUnit decimal.Decimal  `json:"price_per_unit" validate:"required"`
	Currency     string           `json:"currency" validate:"required"`
	Fee          *decimal.Decimal `json:"fee"`
	Description  *string          `json:"description,omitempty"`
}
