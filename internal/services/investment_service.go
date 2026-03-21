package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvestmentServiceInterface interface {
	BackfillInvestmentCashFlows(ctx context.Context, userID int64) error
	FetchInvestmentAssetsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentAsset, *utils.Paginator, error)
	FetchAllInvestmentAssets(ctx context.Context, userID int64) ([]models.InvestmentAsset, error)
	FetchInvestmentAssetByID(ctx context.Context, userID int64, id int64) (*models.InvestmentAsset, error)
	FetchInvestmentTradesPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTrade, *utils.Paginator, error)
	FetchInvestmentTradeByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTrade, error)
	InsertAsset(ctx context.Context, userID int64, req *models.InvestmentAssetReq) (int64, error)
	InsertInvestmentTrade(ctx context.Context, userID int64, req *models.InvestmentTradeReq) (int64, error)
	UpdateInvestmentAsset(ctx context.Context, userID int64, id int64, req *models.InvestmentAssetReq) (int64, error)
	UpdateInvestmentTrade(ctx context.Context, userID int64, id int64, req *models.InvestmentTradeReq) (int64, error)
	DeleteInvestmentAsset(ctx context.Context, userID int64, id int64) error
	DeleteInvestmentTrade(ctx context.Context, userID int64, id int64) error
	GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date *time.Time) (decimal.Decimal, error)
	UpsertAssetPrice(ctx context.Context, tx *gorm.DB, assetID int64, asOf time.Time, price decimal.Decimal, currency string) error
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	SyncAssetPnL(ctx context.Context, userID, assetID int64) error
	SyncAccountPnL(ctx context.Context, userID, accountID int64) error
}

type InvestmentService struct {
	logger           *zap.Logger
	repo             repositories.InvestmentRepositoryInterface
	accRepo          repositories.AccountRepositoryInterface
	settingsRepo     *repositories.SettingsRepository
	loggingRepo      repositories.LoggingRepositoryInterface
	jobDispatcher    queue.JobDispatcher
	priceFetchClient finance.PriceFetcher
}

func NewInvestmentService(
	logger *zap.Logger,
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher queue.JobDispatcher,
	priceFetchClient finance.PriceFetcher,
) *InvestmentService {
	return &InvestmentService{
		logger:           logger,
		repo:             repo,
		accRepo:          accRepo,
		settingsRepo:     settingsRepo,
		jobDispatcher:    jobDispatcher,
		loggingRepo:      loggingRepo,
		priceFetchClient: priceFetchClient,
	}
}

var _ InvestmentServiceInterface = (*InvestmentService)(nil)

// stockTickerRegex is built from finance.ExchangeMap so the two stay in sync.
// Matches US tickers (e.g. AAPL) or TICKER.EXCHANGE (e.g. IWDA.AS).
var stockTickerRegex = func() *regexp.Regexp {
	seen := make(map[string]struct{})
	codes := make([]string, 0)
	for _, code := range finance.ExchangeMap {
		if _, ok := seen[code]; !ok {
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	return regexp.MustCompile(`^[A-Z]{1,7}(\.(` + strings.Join(codes, "|") + `))?$`)
}()

func (s *InvestmentService) FetchInvestmentAssetsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentAsset, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentAssets(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentAssets(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *InvestmentService) FetchAllInvestmentAssets(ctx context.Context, userID int64) ([]models.InvestmentAsset, error) {
	return s.repo.FindAllInvestmentAssets(ctx, nil, userID)
}

func (s *InvestmentService) FetchInvestmentAssetByID(ctx context.Context, userID int64, id int64) (*models.InvestmentAsset, error) {

	record, err := s.repo.FindInvestmentAssetByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) FetchInvestmentTradesPaginated(ctx context.Context, userID int64, p utils.PaginationParams, assetID *int64) ([]models.InvestmentTrade, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentTrades(ctx, nil, userID, p.Filters, assetID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentTrades(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, assetID)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *InvestmentService) FetchInvestmentTradeByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTrade, error) {

	record, err := s.repo.FindInvestmentTradeByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) InsertAsset(ctx context.Context, userID int64, req *models.InvestmentAssetReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	account, err := s.accRepo.FindAccountByID(ctx, tx, req.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find account with given id %w", err)
	}

	// Validate ticker and fetch price
	var currentPrice *decimal.Decimal
	var lastPriceUpdate *time.Time
	var formattedTicker string
	var fetchedPriceCurrency string

	if s.priceFetchClient != nil {

		formattedTicker = strings.ToUpper(req.Ticker)
		switch req.InvestmentType {
		case models.InvestmentCrypto:
			// Ensure format: BTC-USD
			if !strings.Contains(formattedTicker, "-") {
				formattedTicker = formattedTicker + "-USD"
			}

		case models.InvestmentStock, models.InvestmentETF:
			// Allow either:
			//   - pure ticker: AAPL
			//   - ticker + exchange: IWDA.AS
			if !stockTickerRegex.MatchString(formattedTicker) {
				tx.Rollback()
				return 0, fmt.Errorf("invalid stock/ETF ticker: must look like AAPL or IWDA.AS")
			}
		}

		priceData, err := s.priceFetchClient.GetAssetPrice(ctx, formattedTicker, req.InvestmentType)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to fetch price for ticker '%s': %w", formattedTicker, err)
		}

		price := decimal.NewFromFloat(priceData.Price)
		now := time.Unix(priceData.LastUpdate, 0)
		currentPrice = &price
		lastPriceUpdate = &now
		fetchedPriceCurrency = priceData.Currency

	} else {
		// Client not available - allow creation but without price
		currentPrice = nil
		lastPriceUpdate = nil
	}

	hold := models.InvestmentAsset{
		UserID:          userID,
		AccountID:       account.ID,
		InvestmentType:  req.InvestmentType,
		Name:            req.Name,
		Ticker:          formattedTicker,
		Quantity:        req.Quantity,
		Currency:        req.Currency,
		AverageBuyPrice: decimal.Zero,
		CurrentPrice:    currentPrice,
		LastPriceUpdate: lastPriceUpdate,
	}

	holdID, err := s.repo.InsertAsset(ctx, tx, &hold)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if currentPrice != nil && fetchedPriceCurrency != "" {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if err := s.repo.UpsertAssetPrice(ctx, tx, holdID, today, *currentPrice, fetchedPriceCurrency); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to seed price history for new asset: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	quantityString := hold.Quantity.StringFixed(2)

	utils.CompareChanges("", strconv.FormatInt(holdID, 10), changes, "id")
	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", hold.Name, changes, "name")
	utils.CompareChanges("", hold.Ticker, changes, "ticker")
	utils.CompareChanges("", string(hold.InvestmentType), changes, "type")
	utils.CompareChanges("", quantityString, changes, "quantity")

	err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "investment",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return holdID, nil
}

func (s *InvestmentService) fetchCurrentPrice(ctx context.Context, tx *gorm.DB, asset models.InvestmentAsset) (*decimal.Decimal, *time.Time) {
	if s.priceFetchClient == nil {
		return nil, nil
	}

	priceData, err := s.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
	if err != nil {
		return nil, nil
	}

	price := decimal.NewFromFloat(priceData.Price)
	now := time.Unix(priceData.LastUpdate, 0)

	if err := s.repo.UpsertAssetPrice(ctx, nil, asset.ID, now, price, priceData.Currency); err != nil {
		fmt.Printf("warn: failed to upsert asset price history for asset %d: %v\n", asset.ID, err)
	}

	return &price, &now
}

func (s *InvestmentService) InsertInvestmentTrade(ctx context.Context, userID int64, req *models.InvestmentTradeReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, req.AssetID, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find asset with given id %w", err)
	}

	exchangeRate, err := s.GetExchangeRate(ctx, req.Currency, asset.Account.Currency, &req.TxnDate)
	if err != nil {
		return 0, err
	}

	// Validate sell quantity
	if req.TradeType == models.InvestmentSell && req.Quantity.GreaterThan(asset.Quantity) {
		tx.Rollback()
		return 0, fmt.Errorf("cannot sell %s: insufficient quantity (have %s, trying to sell %s)",
			asset.Ticker,
			asset.Quantity.String(),
			req.Quantity.String())
	}

	// Validate buy affordability — balance already reflects cash only
	if req.TradeType == models.InvestmentBuy {
		availableBalance, err := s.accRepo.FindLatestBalance(ctx, tx, asset.AccountID, userID)
		if err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, fmt.Errorf("no balance record found for account")
			}
			return 0, fmt.Errorf("failed to get account balance: %w", err)
		}

		purchaseCost := req.Quantity.Mul(req.PricePerUnit)
		if req.Fee != nil {
			if asset.InvestmentType == models.InvestmentStock || asset.InvestmentType == models.InvestmentETF {
				purchaseCost = purchaseCost.Add(*req.Fee)
			}
		}

		purchaseCostInAccountCurrency := purchaseCost
		if req.Currency != asset.Account.Currency {
			purchaseCostInAccountCurrency = purchaseCost.Mul(exchangeRate)
		}

		// Skip check for zero-cost trades (staking rewards, dividends recorded at price 0).
		if purchaseCostInAccountCurrency.IsPositive() && purchaseCostInAccountCurrency.GreaterThan(availableBalance.EndBalance) {
			tx.Rollback()
			return 0, fmt.Errorf("insufficient funds: need %s %s but only %s %s available",
				purchaseCostInAccountCurrency.StringFixed(2),
				asset.Account.Currency,
				availableBalance.EndBalance.StringFixed(2),
				asset.Account.Currency)
		}
	}

	exchangeRateToUSD, err := s.GetExchangeRate(ctx, req.Currency, "USD", &req.TxnDate)
	if err != nil {
		return 0, err
	}

	fee := decimal.NewFromFloat(0.00)
	if req.Fee != nil {
		fee = *req.Fee
	}

	effectiveQuantity, valueAtBuy := s.calculateTradeValue(req, asset.InvestmentType, fee)
	currentPrice, lastPriceUpdate := s.fetchCurrentPrice(ctx, tx, asset)

	// Calculate trade PnL
	var txnCurrentValue, txnProfitLoss, txnProfitLossPercent, txnRealizedValue decimal.Decimal

	if req.TradeType == models.InvestmentSell {
		if asset.InvestmentType == models.InvestmentCrypto {
			txnRealizedValue = effectiveQuantity.Mul(req.PricePerUnit)
		} else {
			txnRealizedValue = req.Quantity.Mul(req.PricePerUnit).Sub(fee)
		}
		costBasis := asset.AverageBuyPrice.Mul(req.Quantity)
		txnProfitLoss = txnRealizedValue.Sub(costBasis)
		txnCurrentValue, _, _ = s.calculateTradePnL(req.Quantity, currentPrice, costBasis)
		if !costBasis.IsZero() {
			txnProfitLossPercent = txnProfitLoss.Div(costBasis)
		}
	} else {
		txnCurrentValue, txnProfitLoss, txnProfitLossPercent = s.calculateTradePnL(req.Quantity, currentPrice, valueAtBuy)
	}

	txnValueAtBuy := valueAtBuy
	if req.TradeType == models.InvestmentSell {
		txnValueAtBuy = asset.AverageBuyPrice.Mul(req.Quantity)
	}

	txn := models.InvestmentTrade{
		UserID:            userID,
		AssetID:           req.AssetID,
		TxnDate:           req.TxnDate,
		TradeType:         req.TradeType,
		Quantity:          effectiveQuantity,
		PricePerUnit:      req.PricePerUnit,
		Fee:               fee,
		ValueAtBuy:        txnValueAtBuy,
		CurrentValue:      txnCurrentValue,
		RealizedValue:     txnRealizedValue,
		ProfitLoss:        txnProfitLoss,
		ProfitLossPercent: txnProfitLossPercent,
		Currency:          req.Currency,
		ExchangeRateToUSD: exchangeRateToUSD,
		Description:       req.Description,
	}

	txnID, err := s.repo.InsertInvestmentTrade(ctx, tx, &txn)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Write cash flow to balances + update snapshots
	txnDate := req.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txnDate, asset.Account.Currency); err != nil {
		tx.Rollback()
		return 0, err
	}

	if req.TradeType == models.InvestmentBuy {
		// Cash outflow is qty*price for stocks/ETFs — the raw trade cost.
		// valueAtBuy (qty*price-fee) is a separate concept tracking cost basis for PnL;
		// the fee reduces asset value, not the cash paid.
		var cashOut decimal.Decimal
		if asset.InvestmentType == models.InvestmentStock || asset.InvestmentType == models.InvestmentETF {
			cashOut = req.Quantity.Mul(req.PricePerUnit)
		} else {
			cashOut = valueAtBuy
		}
		purchaseCostInAccountCurrency := cashOut
		if req.Currency != asset.Account.Currency {
			purchaseCostInAccountCurrency = cashOut.Mul(exchangeRate)
		}
		if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_outflows", purchaseCostInAccountCurrency); err != nil {
			tx.Rollback()
			return 0, err
		}
	} else {
		// Sell: cash returns via realized P&L
		if err := s.handleSellTrade(ctx, tx, asset, effectiveQuantity, req.PricePerUnit, fee, asset.InvestmentType, txnDate, req.Currency); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, txnDate); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, asset.AccountID, asset.Account.Currency, txnDate, today); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := s.repo.UpdateAssetAfterTrade(ctx, tx, asset.ID, effectiveQuantity, req.PricePerUnit, currentPrice, lastPriceUpdate, req.TradeType, valueAtBuy); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(txnID, 10), changes, "id")
	utils.CompareChanges("", asset.Ticker, changes, "asset")
	utils.CompareChanges("", txn.Quantity.StringFixed(2), changes, "quantity")
	utils.CompareChanges("", txn.PricePerUnit.StringFixed(2), changes, "price_per_unit")
	utils.CompareChanges("", txn.Fee.StringFixed(2), changes, "fee")
	utils.CompareChanges("", txn.TxnDate.UTC().Format(time.RFC3339), changes, "date")
	utils.CompareChanges("", string(txn.TradeType), changes, "type")
	utils.CompareChanges("", txn.Currency, changes, "currency")

	if txn.Description != nil {
		utils.CompareChanges("", *txn.Description, changes, "description")
	}

	err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "investment_trade",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return txnID, nil
}

func (s *InvestmentService) handleSellTrade(ctx context.Context, tx *gorm.DB, asset models.InvestmentAsset, quantitySold, salePrice, fee decimal.Decimal, investmentType models.InvestmentType, txnDate time.Time, tradeCurrency string) error {

	var proceeds decimal.Decimal
	if investmentType == models.InvestmentCrypto {
		proceeds = quantitySold.Mul(salePrice)
	} else {
		proceeds = quantitySold.Mul(salePrice).Sub(fee)
	}

	proceedsInAccountCurrency := proceeds
	if tradeCurrency != asset.Account.Currency {
		exchangeRate, err := s.GetExchangeRate(ctx, tradeCurrency, asset.Account.Currency, &txnDate)
		if err != nil {
			return err
		}
		proceedsInAccountCurrency = proceeds.Mul(exchangeRate)
	}

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txnDate, asset.Account.Currency); err != nil {
		return err
	}

	// Full proceeds return to cash — cost basis was already deducted on the buy
	return s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_inflows", proceedsInAccountCurrency)
}

func (s *InvestmentService) BackfillInvestmentCashFlows(ctx context.Context, userID int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	trades, err := s.repo.FindAllTradesByUserID(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if len(trades) == 0 {
		return tx.Commit().Error
	}

	// Track earliest date per account for frontfill
	earliestByAccount := make(map[int64]time.Time)

	for _, trade := range trades {
		txnDate := trade.TxnDate.UTC().Truncate(24 * time.Hour)

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, trade.Asset.AccountID, txnDate, trade.Asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

		exchangeRate, err := s.GetExchangeRate(ctx, trade.Currency, trade.Asset.Account.Currency, &trade.TxnDate)
		if err != nil {
			tx.Rollback()
			return err
		}

		if trade.TradeType == models.InvestmentBuy {
			// For stocks/ETFs the original raw qty/price/fee are no longer available here,
			// only the stored ValueAtBuy (qty*price-fee). The true cash outflow is qty*price
			// (the raw trade cost), which equals ValueAtBuy + Fee (adding back the fee that was
			// subtracted for cost-basis purposes). Fee reduces asset value, not cash paid.
			var rawCashOut decimal.Decimal
			if trade.Asset.InvestmentType == models.InvestmentStock || trade.Asset.InvestmentType == models.InvestmentETF {
				rawCashOut = trade.ValueAtBuy.Add(trade.Fee)
			} else {
				rawCashOut = trade.ValueAtBuy
			}
			purchaseCost := rawCashOut
			if trade.Currency != trade.Asset.Account.Currency {
				purchaseCost = rawCashOut.Mul(exchangeRate)
			}
			if err := s.accRepo.AddToDailyBalance(ctx, tx, trade.Asset.AccountID, txnDate, "cash_outflows", purchaseCost); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			proceeds := trade.RealizedValue
			if trade.Currency != trade.Asset.Account.Currency {
				proceeds = trade.RealizedValue.Mul(exchangeRate)
			}
			if err := s.accRepo.AddToDailyBalance(ctx, tx, trade.Asset.AccountID, txnDate, "cash_inflows", proceeds); err != nil {
				tx.Rollback()
				return err
			}
		}

		if earliest, ok := earliestByAccount[trade.Asset.AccountID]; !ok || txnDate.Before(earliest) {
			earliestByAccount[trade.Asset.AccountID] = txnDate
		}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for accountID, earliestDate := range earliestByAccount {
		// Need currency — grab from first trade for this account
		var currency string
		for _, t := range trades {
			if t.Asset.AccountID == accountID {
				currency = t.Asset.Account.Currency
				break
			}
		}

		if err := s.accRepo.FrontfillBalances(ctx, tx, accountID, currency, earliestDate); err != nil {
			tx.Rollback()
			return err
		}

		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accountID, currency, earliestDate, today); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *InvestmentService) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date *time.Time) (decimal.Decimal, error) {
	if fromCurrency == toCurrency {
		return decimal.NewFromFloat(1.0), nil
	}

	if s.priceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}

	var rate float64
	var err error

	if date != nil {
		rate, err = s.priceFetchClient.GetExchangeRateOnDate(ctx, fromCurrency, toCurrency, *date)
	} else {
		rate, err = s.priceFetchClient.GetExchangeRate(ctx, fromCurrency, toCurrency)
	}

	if err != nil {
		return decimal.Zero, err
	}

	return decimal.NewFromFloat(rate), nil
}

func (s *InvestmentService) calculateTradeValue(req *models.InvestmentTradeReq, investmentType models.InvestmentType, fee decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	effectiveQuantity := req.Quantity
	var valueAtBuy decimal.Decimal

	if investmentType == models.InvestmentCrypto {
		// Crypto: (quantity - fee) * price_per_unit
		effectiveQuantity = req.Quantity.Sub(fee)
		valueAtBuy = effectiveQuantity.Mul(req.PricePerUnit)
	} else {
		// Stock/ETF: (quantity * price_per_unit) - fee
		valueAtBuy = req.Quantity.Mul(req.PricePerUnit).Sub(fee)
	}

	return effectiveQuantity, valueAtBuy
}

func (s *InvestmentService) calculateTradePnL(quantity decimal.Decimal, currentPrice *decimal.Decimal, valueAtBuy decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	if currentPrice != nil && !currentPrice.IsZero() {
		currentValue := quantity.Mul(*currentPrice)
		profitLoss := currentValue.Sub(valueAtBuy)

		var profitLossPercent decimal.Decimal
		if !valueAtBuy.IsZero() {
			profitLossPercent = profitLoss.Div(valueAtBuy)
		}

		return currentValue, profitLoss, profitLossPercent
	}

	return decimal.Zero, decimal.Zero, decimal.Zero
}

func (s *InvestmentService) UpdateInvestmentAsset(ctx context.Context, userID int64, id int64, req *models.InvestmentAssetReq) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load existing asset
	exHold, err := s.repo.FindInvestmentAssetByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find asset: %w", err)
	}

	hold := models.InvestmentAsset{
		ID:     exHold.ID,
		UserID: userID,
		Name:   req.Name,
	}

	holdID, err := s.repo.UpdateInvestmentAsset(ctx, tx, hold)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(holdID, 10), changes, "id")
	utils.CompareChanges("", exHold.Ticker, changes, "asset")
	utils.CompareChanges(exHold.Name, hold.Name, changes, "name")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "investment",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return holdID, nil
}

func (s *InvestmentService) UpdateInvestmentTrade(ctx context.Context, userID int64, id int64, req *models.InvestmentTradeReq) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load existing record
	exTxn, err := s.repo.FindInvestmentTradeByID(ctx, tx, id, userID)
	if err != nil {
		return 0, fmt.Errorf("can't find investment trade with given id %w", err)
	}

	// Load existing relations
	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, exTxn.AssetID, userID)
	if err != nil {
		return 0, fmt.Errorf("can't find existing asset: %w", err)
	}

	txn := models.InvestmentTrade{
		ID:          exTxn.ID,
		UserID:      userID,
		AssetID:     asset.ID,
		Description: req.Description,
	}

	txnID, err := s.repo.UpdateInvestmentTrade(ctx, tx, txn)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()

	oldDesc := ""
	if exTxn.Description != nil {
		oldDesc = *exTxn.Description
	}

	newDesc := ""
	if txn.Description != nil {
		newDesc = *txn.Description
	}

	if oldDesc != newDesc {
		utils.CompareChanges("", strconv.FormatInt(txnID, 10), changes, "id")
		utils.CompareChanges("", asset.Ticker, changes, "asset")
		utils.CompareChanges(oldDesc, newDesc, changes, "description")
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "investment_trade",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return txnID, nil
}

func (s *InvestmentService) DeleteInvestmentAsset(ctx context.Context, userID int64, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find asset: %w", err)
	}

	earliestTxnDate, err := s.repo.GetEarliestTradeDate(ctx, tx, id, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return err
	}

	// Reverse all cash flows written by trades for this asset
	allTrades, err := s.repo.FindAllTradesByAssetID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, trade := range allTrades {
		txnDate := trade.TxnDate.UTC().Truncate(24 * time.Hour)

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txnDate, asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

		exchangeRate, err := s.GetExchangeRate(ctx, trade.Currency, asset.Account.Currency, &trade.TxnDate)
		if err != nil {
			tx.Rollback()
			return err
		}

		if trade.TradeType == models.InvestmentBuy {
			// Reverse cash outflow
			purchaseCost := trade.ValueAtBuy
			if trade.Currency != asset.Account.Currency {
				purchaseCost = trade.ValueAtBuy.Mul(exchangeRate)
			}
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_outflows", purchaseCost.Neg()); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			// Reverse sell: subtract the proceeds that were credited to cash
			proceeds := trade.RealizedValue
			proceedsInAccountCurrency := proceeds
			if trade.Currency != asset.Account.Currency {
				proceedsInAccountCurrency = proceeds.Mul(exchangeRate)
			}
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_inflows", proceedsInAccountCurrency.Neg()); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Bulk delete all trades for this asset
	if err := s.repo.DeleteAllTradesForAsset(ctx, tx, id, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete the asset
	if err := s.repo.DeleteInvestmentAsset(ctx, tx, id); err != nil {
		tx.Rollback()
		return err
	}

	// Rebuild balances and snapshots from the earliest trade date
	if !earliestTxnDate.IsZero() {
		today := time.Now().UTC().Truncate(24 * time.Hour)

		if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, earliestTxnDate); err != nil {
			tx.Rollback()
			return err
		}

		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, asset.AccountID, asset.Account.Currency, earliestTxnDate, today); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(asset.Ticker, "", changes, "ticker")
	utils.CompareChanges(asset.Name, "", changes, "name")

	err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "investment",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *InvestmentService) DeleteInvestmentTrade(ctx context.Context, userID int64, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	exTxn, err := s.repo.FindInvestmentTradeByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find investment trade: %w", err)
	}

	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, exTxn.AssetID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find asset: %w", err)
	}

	// Validate deletion: check if removing this buy would cause negative quantity
	if exTxn.TradeType == models.InvestmentBuy {
		newQuantity := asset.Quantity.Sub(exTxn.Quantity)
		if newQuantity.LessThan(decimal.Zero) {
			tx.Rollback()
			return fmt.Errorf("cannot delete buy trade: would result in negative quantity (current: %s, removing: %s)",
				asset.Quantity.String(),
				exTxn.Quantity.String())
		}
	}

	txnDate := exTxn.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txnDate, asset.Account.Currency); err != nil {
		tx.Rollback()
		return err
	}

	// Reverse the cash flow that was written when the trade was created
	if exTxn.TradeType == models.InvestmentBuy {
		// Reverse cash outflow — cash comes back
		exchangeRate, err := s.GetExchangeRate(ctx, exTxn.Currency, asset.Account.Currency, &exTxn.TxnDate)
		if err != nil {
			tx.Rollback()
			return err
		}
		purchaseCost := exTxn.ValueAtBuy
		if exTxn.Currency != asset.Account.Currency {
			purchaseCost = exTxn.ValueAtBuy.Mul(exchangeRate)
		}
		if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_outflows", purchaseCost.Neg()); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Reverse sell: subtract the proceeds that were credited to cash
		proceeds := exTxn.RealizedValue
		proceedsInAccountCurrency := proceeds
		if exTxn.Currency != asset.Account.Currency {
			exchangeRate, err := s.GetExchangeRate(ctx, exTxn.Currency, asset.Account.Currency, &exTxn.TxnDate)
			if err != nil {
				tx.Rollback()
				return err
			}
			proceedsInAccountCurrency = proceeds.Mul(exchangeRate)
		}
		if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_inflows", proceedsInAccountCurrency.Neg()); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := s.repo.DeleteInvestmentTrade(ctx, tx, id); err != nil {
		tx.Rollback()
		return err
	}

	// Recalculate asset from remaining trades
	if err := s.repo.RecalculateAssetFromTrades(ctx, tx, asset.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, exTxn.TxnDate); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(
		ctx, tx,
		userID,
		asset.AccountID,
		asset.Account.Currency,
		exTxn.TxnDate.UTC().Truncate(24*time.Hour),
		today,
	); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(asset.Ticker, "", changes, "asset")
	utils.CompareChanges(exTxn.Quantity.StringFixed(2), "", changes, "quantity")
	utils.CompareChanges(exTxn.PricePerUnit.StringFixed(2), "", changes, "price_per_unit")
	utils.CompareChanges(string(exTxn.TradeType), "", changes, "type")

	err = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "investment_trade",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *InvestmentService) UpsertAssetPrice(ctx context.Context, tx *gorm.DB, assetID int64, asOf time.Time, price decimal.Decimal, currency string) error {
	return s.repo.UpsertAssetPrice(ctx, tx, assetID, asOf, price, currency)
}

func (s *InvestmentService) RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error {
	return s.repo.RecalculateAssetFromTrades(ctx, nil, assetID, userID)
}

func (s *InvestmentService) GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error) {
	return s.repo.GetAssetIDsForAccount(ctx, nil, accountID, userID)
}

func (s *InvestmentService) UpdateSnapshotMarketValues(ctx context.Context, userID int64) error {
	return s.accRepo.UpdateSnapshotMarketValues(ctx, nil, userID)
}

func (s *InvestmentService) SyncAssetPnL(ctx context.Context, userID, assetID int64) error {
	return s.jobDispatcher.Dispatch(queue.NewRecalculateAssetPnLJob(
		s.logger.Named("pnl_sync"),
		s, userID, &assetID, nil,
	))
}

func (s *InvestmentService) SyncAccountPnL(ctx context.Context, userID, accountID int64) error {
	return s.jobDispatcher.Dispatch(queue.NewRecalculateAssetPnLJob(
		s.logger.Named("pnl_sync"),
		s, userID, nil, &accountID,
	))
}
