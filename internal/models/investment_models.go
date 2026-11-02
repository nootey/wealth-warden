package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type (
	InvestmentType string
	TradeType      string
	IncomeType     string
)

const (
	InvestmentStock  InvestmentType = "stock"
	InvestmentETF    InvestmentType = "etf"
	InvestmentCrypto InvestmentType = "crypto"

	InvestmentBuy  TradeType = "buy"
	InvestmentSell TradeType = "sell"

	IncomeTypeStaking  IncomeType = "staking_reward"
	IncomeTypeDividend IncomeType = "dividend"
)

type AssetBackfillRow struct {
	Ticker         string
	InvestmentType InvestmentType
	Currency       string
	EarliestTrade  time.Time
	LastPriceDate  *time.Time
}

type AssetPriceSyncRow struct {
	Ticker         string
	InvestmentType InvestmentType
}

type CurrencyPair struct {
	FromCurrency string
	ToCurrency   string
}

type AssetPriceChange struct {
	AssetID  int64
	UserID   int64
	Ticker   string
	OldPrice *decimal.Decimal
	NewPrice decimal.Decimal
}

type InvestmentAsset struct {
	ID                  int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID           int64            `gorm:"not null;index:idx_assets_account" json:"account_id"`
	UserID              int64            `gorm:"not null;index:idx_assets_user" json:"user_id"`
	InvestmentType      InvestmentType   `gorm:"type:investment_type;not null" json:"investment_type"`
	Name                string           `gorm:"type:varchar(255);not null" json:"name"`
	Ticker              string           `gorm:"type:varchar(20);not null;index:idx_assets_ticker" json:"ticker"`
	Quantity            decimal.Decimal  `gorm:"type:decimal(19,8);not null" json:"quantity"`
	AverageBuyPrice     decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"average_buy_price"`
	ValueAtBuy          decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"value_at_buy"`
	TotalFees           decimal.Decimal  `gorm:"type:decimal(19,6);not null;default:0" json:"total_fees"`
	CurrentValue        decimal.Decimal  `gorm:"->" json:"current_value"`
	ProfitLoss          decimal.Decimal  `gorm:"->" json:"profit_loss"`
	ProfitLossPercent   decimal.Decimal  `gorm:"->" json:"profit_loss_percent"`
	LatestPrice         *decimal.Decimal `gorm:"->" json:"latest_price"`
	LatestPriceCurrency *string          `gorm:"->" json:"latest_price_currency"`
	LastPriceUpdate     *time.Time       `gorm:"->" json:"last_price_update"`
	Currency            string           `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	Account             Account          `json:"account"`
	ImportID            *int64           `json:"import_id,omitempty"`
	TaxSummary          *AssetTaxSummary `gorm:"-" json:"tax_summary,omitempty"`
	CreatedAt           time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

type InvestmentTrade struct {
	ID                int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int64           `gorm:"not null" json:"user_id"`
	AssetID           int64           `gorm:"not null;index:idx_inv_trans_asset" json:"asset_id"`
	TxnDate           time.Time       `gorm:"type:date;not null;index:idx_inv_trans_date" json:"txn_date"`
	TradeType         TradeType       `gorm:"type:varchar(4);not null" json:"trade_type"`
	Quantity          decimal.Decimal `gorm:"type:decimal(19,8);not null" json:"quantity"`
	Fee               decimal.Decimal `gorm:"type:decimal(19,6);not null;default:0" json:"fee"`
	PricePerUnit      decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"price_per_unit"`
	ValueAtBuy        decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"value_at_buy"`
	RealizedValue     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"realized_value"`
	CurrentValue      decimal.Decimal `gorm:"->" json:"current_value"`
	ProfitLoss        decimal.Decimal `gorm:"->" json:"profit_loss"`
	ProfitLossPercent decimal.Decimal `gorm:"->" json:"profit_loss_percent"`
	Currency          string          `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	ExchangeRateToUSD decimal.Decimal `gorm:"type:decimal(19,6);not null;default:1.0" json:"exchange_rate_to_usd"`
	Description       *string         `gorm:"type:varchar(255)" json:"description"`
	TransactionID     *int64          `json:"transaction_id,omitempty"`
	Asset             InvestmentAsset `json:"asset"`
	ImportID          *int64          `json:"import_id,omitempty"`
	TaxInfo           *TradeTaxInfo   `gorm:"-" json:"tax_info,omitempty"`
	CreatedAt         time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// amount must already be in the account's currency.
func NewTradeCashTransaction(userID, accountID int64, categoryID *int64, ticker, currency string, tradeType TradeType, txnDate time.Time, amount decimal.Decimal) Transaction {
	txnType := TxnDirectionExpense
	desc := "Buy: " + ticker
	if tradeType == InvestmentSell {
		txnType = TxnDirectionIncome
		desc = "Sell: " + ticker
	}

	return Transaction{
		UserID:          userID,
		AccountID:       accountID,
		CategoryID:      categoryID,
		Direction:       txnType,
		Amount:          amount,
		Currency:        currency,
		TxnDate:         txnDate,
		Description:     &desc,
		TransactionType: TxnTypeTrade,
	}
}

type TickerPriceHistory struct {
	Ticker    string          `gorm:"primaryKey;type:varchar(20)" json:"ticker"`
	AsOf      time.Time       `gorm:"type:date;primaryKey" json:"as_of"`
	Price     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"price"`
	Currency  string          `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (TickerPriceHistory) TableName() string {
	return "ticker_price_history"
}

type ExchangeRateHistory struct {
	FromCurrency string          `gorm:"primaryKey;type:char(3)" json:"from_currency"`
	ToCurrency   string          `gorm:"primaryKey;type:char(3)" json:"to_currency"`
	AsOf         time.Time       `gorm:"primaryKey;type:date" json:"as_of"`
	Rate         decimal.Decimal `gorm:"type:decimal(19,6);not null" json:"rate"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (ExchangeRateHistory) TableName() string {
	return "exchange_rate_history"
}

func (InvestmentIncome) TableName() string {
	return "investment_income"
}

type InvestmentIncome struct {
	ID                  int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID              int64            `gorm:"not null;index:idx_inv_income_user" json:"user_id"`
	AssetID             int64            `gorm:"not null;index:idx_inv_income_asset" json:"asset_id"`
	TxnDate             time.Time        `gorm:"type:date;not null;index:idx_inv_income_date" json:"txn_date"`
	IncomeType          IncomeType       `gorm:"type:income_type;not null" json:"income_type"`
	Quantity            *decimal.Decimal `gorm:"type:decimal(19,8)" json:"quantity"`
	Amount              decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"amount"`
	TaxWithheld         *decimal.Decimal `gorm:"type:decimal(19,4)" json:"tax_withheld"`
	Currency            string           `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	ExchangeRateToUSD   decimal.Decimal  `gorm:"type:decimal(19,6);not null;default:1.0" json:"exchange_rate_to_usd"`
	Notes               *string          `gorm:"type:varchar(255)" json:"notes"`
	LinkedTransactionID *int64           `gorm:"index" json:"linked_transaction_id,omitempty"`
	Asset               InvestmentAsset  `json:"asset"`
	CreatedAt           time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

var InvestmentAssetSort = SortSpec[InvestmentAsset]{
	Source: "investment_assets",
	Fields: map[string]SortKey[InvestmentAsset]{
		"name":              {Column: "investment_assets.name", Value: func(a InvestmentAsset) any { return a.Name }},
		"ticker":            {Column: "investment_assets.ticker", Value: func(a InvestmentAsset) any { return a.Ticker }},
		"quantity":          {Column: "investment_assets.quantity", Value: func(a InvestmentAsset) any { return a.Quantity }},
		"average_buy_price": {Column: "investment_assets.average_buy_price", Value: func(a InvestmentAsset) any { return a.AverageBuyPrice }},
		"value_at_buy":      {Column: "investment_assets.value_at_buy", Value: func(a InvestmentAsset) any { return a.ValueAtBuy }},
		"current_value":     {Column: "investment_assets.current_value", Value: func(a InvestmentAsset) any { return a.CurrentValue }},
		"profit_loss":       {Column: "investment_assets.profit_loss", Value: func(a InvestmentAsset) any { return a.ProfitLoss }},
		"created_at":        {Column: "investment_assets.created_at", Value: func(a InvestmentAsset) any { return a.CreatedAt }},
		"account":           {Column: "accounts.name", Value: func(a InvestmentAsset) any { return a.Account.Name }},
	},
	ID:      SortKey[InvestmentAsset]{Column: "investment_assets.id", Value: func(a InvestmentAsset) any { return a.ID }},
	Default: "created_at",
}

var InvestmentTradeSort = SortSpec[InvestmentTrade]{
	Source: "investment_trades",
	Fields: map[string]SortKey[InvestmentTrade]{
		"txn_date":       {Column: "investment_trades.txn_date", Value: func(t InvestmentTrade) any { return t.TxnDate }},
		"trade_type":     {Column: "investment_trades.trade_type", Value: func(t InvestmentTrade) any { return t.TradeType }},
		"quantity":       {Column: "investment_trades.quantity", Value: func(t InvestmentTrade) any { return t.Quantity }},
		"price_per_unit": {Column: "investment_trades.price_per_unit", Value: func(t InvestmentTrade) any { return t.PricePerUnit }},
		"value_at_buy":   {Column: "investment_trades.value_at_buy", Value: func(t InvestmentTrade) any { return t.ValueAtBuy }},
		"current_value":  {Column: "investment_trades.current_value", Value: func(t InvestmentTrade) any { return t.CurrentValue }},
		"profit_loss":    {Column: "investment_trades.profit_loss", Value: func(t InvestmentTrade) any { return t.ProfitLoss }},
		"created_at":     {Column: "investment_trades.created_at", Value: func(t InvestmentTrade) any { return t.CreatedAt }},
		"asset.name":     {Column: "investment_assets.name", Value: func(t InvestmentTrade) any { return t.Asset.Name }},
		"asset.ticker":   {Column: "investment_assets.ticker", Value: func(t InvestmentTrade) any { return t.Asset.Ticker }},
	},
	ID:      SortKey[InvestmentTrade]{Column: "investment_trades.id", Value: func(t InvestmentTrade) any { return t.ID }},
	Default: "txn_date",
}

var InvestmentIncomeSort = SortSpec[InvestmentIncome]{
	Fields: map[string]SortKey[InvestmentIncome]{
		"txn_date":     {Column: "investment_income.txn_date", Value: func(i InvestmentIncome) any { return i.TxnDate }},
		"income_type":  {Column: "investment_income.income_type", Value: func(i InvestmentIncome) any { return i.IncomeType }},
		"quantity":     {Column: "investment_income.quantity", Value: func(i InvestmentIncome) any { return i.Quantity }},
		"amount":       {Column: "investment_income.amount", Value: func(i InvestmentIncome) any { return i.Amount }},
		"tax_withheld": {Column: "investment_income.tax_withheld", Value: func(i InvestmentIncome) any { return i.TaxWithheld }},
		"notes":        {Column: "investment_income.notes", Value: func(i InvestmentIncome) any { return i.Notes }},
		"created_at":   {Column: "investment_income.created_at", Value: func(i InvestmentIncome) any { return i.CreatedAt }},
	},
	ID:      SortKey[InvestmentIncome]{Column: "investment_income.id", Value: func(i InvestmentIncome) any { return i.ID }},
	Default: "txn_date",
}

type TradeTaxInfo struct {
	DaysHeld             int              `json:"days_held"`
	TaxablePercent       *decimal.Decimal `json:"taxable_percent"`
	TaxableProfit        decimal.Decimal  `json:"taxable_profit"`
	DaysUntilNextBracket *int             `json:"days_until_next_bracket"`
	DaysUntilTaxFree     *int             `json:"days_until_tax_free"`
}

type AssetTaxSummary struct {
	EstimatedTaxDue decimal.Decimal  `json:"estimated_tax_due"`
	AfterTaxPnL     decimal.Decimal  `json:"after_tax_pnl"`
	TaxablePercent  *decimal.Decimal `json:"taxable_percent"`
}

type InvestmentTaxBracket struct {
	ID             int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64           `gorm:"not null;index:idx_tax_brackets_user" json:"user_id"`
	InvestmentType InvestmentType  `gorm:"type:investment_type;not null" json:"investment_type"`
	MinDaysHeld    int             `gorm:"not null" json:"min_days_held"`
	ToDays         *int            `json:"to_days"`
	TaxablePercent decimal.Decimal `gorm:"type:decimal(5,2);not null" json:"taxable_percent"`
	Label          *string         `gorm:"type:varchar(100)" json:"label"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

type InvestmentTaxSettings struct {
	ID                    int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                int64     `gorm:"not null;uniqueIndex" json:"user_id"`
	LossOffsettingEnabled bool      `gorm:"not null;default:false" json:"loss_offsetting_enabled"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type InvestmentTaxBracketReq struct {
	InvestmentType InvestmentType  `json:"investment_type" validate:"required"`
	MinDaysHeld    int             `json:"min_days_held" validate:"min=0"`
	ToDays         *int            `json:"to_days,omitempty"`
	TaxablePercent decimal.Decimal `json:"taxable_percent" validate:"required"`
	Label          *string         `json:"label,omitempty"`
}

type InvestmentTaxSettingsReq struct {
	LossOffsettingEnabled bool `json:"loss_offsetting_enabled"`
}

type InvestmentTaxBracketsCopyReq struct {
	FromType InvestmentType `json:"from_type" validate:"required"`
	ToType   InvestmentType `json:"to_type" validate:"required"`
}

type InvestmentIncomeReq struct {
	AssetID     int64            `json:"asset_id" validate:"required"`
	TxnDate     time.Time        `json:"txn_date" validate:"required"`
	IncomeType  IncomeType       `json:"income_type" validate:"required"`
	Quantity    *decimal.Decimal `json:"quantity"`
	Amount      *decimal.Decimal `json:"amount,omitempty"`
	TaxWithheld *decimal.Decimal `json:"tax_withheld,omitempty"`
	Currency    string           `json:"currency" validate:"required"`
	Notes       *string          `json:"notes,omitempty"`
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

type AllocationAssetRow struct {
	Ticker         string
	Name           string
	InvestmentType InvestmentType
	SourceCurrency string
	AccountID      int64
	AccountName    string
	Priced         bool
	ValueText      string
}

type AllocationRow struct {
	Key    string          `json:"key"`
	Label  string          `json:"label"`
	Value  decimal.Decimal `json:"value"`
	Weight decimal.Decimal `json:"weight"`
}

type PortfolioAllocation struct {
	Currency       string                     `json:"currency"`
	TotalValue     decimal.Decimal            `json:"total_value"`
	UnpricedAssets int                        `json:"unpriced_assets"`
	Groups         map[string][]AllocationRow `json:"groups"`
}

type ReturnFlowRow struct {
	AssetID           int64
	Ticker            string
	Name              string
	FlowDate          time.Time
	AmountText        string
	DisplayAmountText string
	IsTerminal        bool
	HeldUnpriced      bool
}

type PortfolioReturnRow struct {
	Key          string           `json:"key"`
	Label        string           `json:"label"`
	Rate         *decimal.Decimal `json:"rate"`
	Reason       string           `json:"reason,omitempty"`
	CurrentValue decimal.Decimal  `json:"current_value"`
}

type PortfolioReturns struct {
	Currency       string               `json:"currency"`
	UnpricedAssets int                  `json:"unpriced_assets"`
	Portfolio      PortfolioReturnRow   `json:"portfolio"`
	Assets         []PortfolioReturnRow `json:"assets"`
}
