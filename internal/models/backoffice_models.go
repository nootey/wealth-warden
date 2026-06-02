package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type ZeroCostTradePreview struct {
	ID       int64           `json:"id"`
	TxnDate  time.Time       `json:"txn_date"`
	Quantity decimal.Decimal `json:"quantity"`
	Currency string          `json:"currency"`
}

type ZeroCostTradeAssetGroup struct {
	AssetID        int64              `json:"asset_id"`
	Ticker         string             `json:"ticker"`
	AssetName      string             `json:"asset_name"`
	InvestmentType InvestmentType     `json:"investment_type"`
	IncomeType     IncomeType         `json:"income_type"`
	TradeCount     int                `json:"trade_count"`
	Trades         []ZeroCostTradePreview `json:"trades"`
}

type ZeroCostTradeMigrationPreview struct {
	TotalTrades int                       `json:"total_trades"`
	AssetCount  int                       `json:"asset_count"`
	Assets      []ZeroCostTradeAssetGroup `json:"assets"`
}

type ZeroCostMigrationError struct {
	AssetID int64  `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}

type ZeroCostMigrationResult struct {
	TotalProcessed  int                      `json:"total_processed"`
	AssetsProcessed int                      `json:"assets_processed"`
	AssetsFailed    int                      `json:"assets_failed"`
	Errors          []ZeroCostMigrationError `json:"errors,omitempty"`
}
