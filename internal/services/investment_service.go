package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const priceBackfillFlushSize = 100

type InvestmentServiceInterface interface {
	RebuildInvestmentDerivedData(ctx context.Context, userID int64) error
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
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
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
	GetAssetsForPriceBackfill(ctx context.Context) ([]models.AssetBackfillRow, error)
	GetTickersForPriceSync(ctx context.Context) ([]models.AssetPriceSyncRow, error)
	ApplyTickerPrice(ctx context.Context, ticker string, price decimal.Decimal, currency string, now time.Time) ([]models.AssetPriceChange, error)
	GetActiveCurrencyPairs(ctx context.Context) ([]models.CurrencyPair, error)
	GetUserIDsWithActiveInvestments(ctx context.Context) ([]int64, error)
	BackfillAssetPriceHistory(ctx context.Context, assetID int64, ticker string, from, to time.Time) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64) error
	CreateInvestmentIncome(ctx context.Context, userID int64, req *models.InvestmentIncomeReq) (int64, error)
	DeleteInvestmentIncome(ctx context.Context, userID int64, id int64) error
	FetchInvestmentIncomeByAsset(ctx context.Context, userID int64, assetID int64, p utils.PaginationParams) ([]models.InvestmentIncome, *utils.Paginator, error)
	MigrateZeroCostTradesForAsset(ctx context.Context, userID, assetID int64, trades []models.InvestmentTrade) error
	FetchTaxBrackets(ctx context.Context, userID int64) ([]models.InvestmentTaxBracket, error)
	InsertTaxBracket(ctx context.Context, userID int64, req *models.InvestmentTaxBracketReq) (int64, error)
	UpdateTaxBracket(ctx context.Context, userID int64, id int64, req *models.InvestmentTaxBracketReq) error
	DeleteTaxBracket(ctx context.Context, userID int64, id int64) error
	FetchTaxSettings(ctx context.Context, userID int64) (models.InvestmentTaxSettings, error)
	SaveTaxSettings(ctx context.Context, userID int64, req *models.InvestmentTaxSettingsReq) error
	CopyTaxBrackets(ctx context.Context, userID int64, fromType, toType models.InvestmentType) error
	FetchPortfolioAllocation(ctx context.Context, userID int64, currency string) (*models.PortfolioAllocation, error)
}

type InvestmentService struct {
	logger           *zap.Logger
	repo             repositories.InvestmentRepositoryInterface
	accRepo          repositories.AccountRepositoryInterface
	txnRepo          repositories.TransactionRepositoryInterface
	settingsRepo     *repositories.SettingsRepository
	jobDispatcher    jobqueue.Dispatcher
	priceFetchClient finance.PriceFetcher
}

func NewInvestmentService(
	logger *zap.Logger,
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
	jobDispatcher jobqueue.Dispatcher,
	priceFetchClient finance.PriceFetcher,
) *InvestmentService {
	return &InvestmentService{
		logger:           logger,
		repo:             repo,
		accRepo:          accRepo,
		txnRepo:          txnRepo,
		settingsRepo:     settingsRepo,
		jobDispatcher:    jobDispatcher,
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

	brackets, err := s.repo.FindTaxBracketsByUserAndType(ctx, nil, userID, record.InvestmentType)
	if err != nil {
		return nil, err
	}
	if len(brackets) > 0 {
		settings, err := s.repo.FindTaxSettings(ctx, nil, userID)
		if err != nil {
			return nil, err
		}
		trades, err := s.repo.FindAllTradesByAssetID(ctx, nil, record.ID, userID)
		if err != nil {
			return nil, err
		}
		summary := utils.ComputeAssetTaxSummary(record, trades, brackets, settings, time.Now().UTC())
		record.TaxSummary = &summary
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

	brackets, err := s.repo.FindTaxBracketsByUserAndType(ctx, nil, userID, record.Asset.InvestmentType)
	if err != nil {
		return nil, err
	}
	if len(brackets) > 0 {
		if record.TradeType == models.InvestmentBuy {
			info := utils.ComputeBuyTradeTaxInfo(record, brackets, time.Now().UTC())
			record.TaxInfo = &info
		} else {
			allTrades, err := s.repo.FindAllTradesByAssetID(ctx, nil, record.AssetID, userID)
			if err != nil {
				return nil, err
			}
			info := utils.ComputeSellTradeTaxInfo(record, allTrades, brackets)
			record.TaxInfo = &info
		}
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
		if err := s.repo.UpsertAssetPrice(ctx, tx, []models.AssetPriceHistory{{AssetID: holdID, AsOf: today, Price: *currentPrice, Currency: fetchedPriceCurrency}}); err != nil {
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

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "create",
		Category:    "investment_asset",
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

	if err := s.repo.UpsertAssetPrice(ctx, nil, []models.AssetPriceHistory{{AssetID: asset.ID, AsOf: now, Price: price, Currency: priceData.Currency}}); err != nil {
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

	// The full req.Quantity leaves holdings on a sell; the fee (in coin units for
	// crypto) only reduces cash proceeds, it does not stay in the position.
	holdingsQuantity := effectiveQuantity
	if req.TradeType == models.InvestmentSell {
		holdingsQuantity = req.Quantity
	}

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
		basis := valueAtBuy
		if asset.InvestmentType != models.InvestmentCrypto {
			basis = valueAtBuy.Add(fee)
		}
		txnCurrentValue, txnProfitLoss, txnProfitLossPercent = s.calculateTradePnL(effectiveQuantity, currentPrice, basis)
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
		Quantity:          holdingsQuantity,
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
		grossCost := valueAtBuy
		if asset.InvestmentType != models.InvestmentCrypto {
			grossCost = grossCost.Add(fee)
		}
		purchaseCostInAccountCurrency := grossCost
		if req.Currency != asset.Account.Currency {
			purchaseCostInAccountCurrency = grossCost.Mul(exchangeRate)
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

	if err := s.repo.UpdateAssetAfterTrade(ctx, tx, asset.ID, holdingsQuantity, req.PricePerUnit, currentPrice, lastPriceUpdate, req.TradeType, valueAtBuy, fee); err != nil {
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

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "create",
		Category:    "investment_trade",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.SyncAssetAfterTradeArgs{
		UserID:         userID,
		AssetID:        asset.ID,
		Ticker:         asset.Ticker,
		InvestmentType: asset.InvestmentType,
		TradeDate:      txnDate,
	}); err != nil {
		s.logger.Warn("Failed to dispatch post-trade sync job", zap.Error(err))
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

func (s *InvestmentService) resolveUserTradeRates(ctx context.Context, userID int64) (map[utils.TradeExchangeRateKey]decimal.Decimal, error) {
	trades, err := s.repo.FindAllTradesByUserID(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	rates := make(map[utils.TradeExchangeRateKey]decimal.Decimal)
	for _, trade := range trades {
		if trade.Currency == trade.Asset.Account.Currency {
			continue
		}
		key := utils.NewTradeExchangeRateKey(trade)
		if _, ok := rates[key]; ok {
			continue
		}
		rate, err := s.GetExchangeRate(ctx, key.From, key.To, &trade.TxnDate)
		if err != nil {
			return nil, err
		}
		rates[key] = rate
	}
	return rates, nil
}

func (s *InvestmentService) RebuildInvestmentDerivedData(ctx context.Context, userID int64) error {
	rates, err := s.resolveUserTradeRates(ctx, userID)
	if err != nil {
		return err
	}

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

	if err := s.rebuildDerivedData(ctx, tx, userID, trades, rates); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InvestmentService) CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error {
	// The correction changes value_at_buy only, so the uncorrected trades name
	// the same rates.
	rates, err := s.resolveUserTradeRates(ctx, userID)
	if err != nil {
		return err
	}

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

	if err := s.correctTradeFeeAccounting(ctx, tx, userID); err != nil {
		tx.Rollback()
		return err
	}

	corrected, err := s.repo.FindAllTradesByUserID(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.rebuildDerivedData(ctx, tx, userID, corrected, rates); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InvestmentService) rebuildDerivedData(
	ctx context.Context,
	tx *gorm.DB,
	userID int64,
	trades []models.InvestmentTrade,
	rates map[utils.TradeExchangeRateKey]decimal.Decimal,
) error {
	if err := s.accRepo.ClearInvestmentCashFlows(ctx, tx, userID); err != nil {
		return err
	}

	if err := s.accRepo.ClearInvestmentSnapshots(ctx, tx, userID); err != nil {
		return err
	}

	if err := s.addTradeCashFlows(ctx, tx, userID, trades, rates); err != nil {
		return err
	}

	if err := s.rebuildSnapshots(ctx, tx, userID); err != nil {
		return err
	}

	// Must take the tx: on its own connection this blocks on rows the open
	// transaction holds, and that transaction waits for it to return.
	return s.accRepo.UpdateSnapshotMarketValues(ctx, tx, userID, nil)
}

func (s *InvestmentService) addTradeCashFlows(
	ctx context.Context,
	tx *gorm.DB,
	userID int64,
	trades []models.InvestmentTrade,
	rates map[utils.TradeExchangeRateKey]decimal.Decimal,
) error {
	if len(trades) == 0 {
		return nil
	}

	// Track earliest date and currency per account for frontfill
	earliestByAccount := make(map[int64]time.Time)
	accountCurrency := make(map[int64]string)

	for _, trade := range trades {
		txnDate := trade.TxnDate.UTC().Truncate(24 * time.Hour)
		accCurrency := trade.Asset.Account.Currency

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, trade.Asset.AccountID, txnDate, accCurrency); err != nil {
			return err
		}

		field := "cash_inflows"
		amount := trade.RealizedValue

		if trade.TradeType == models.InvestmentBuy {
			field = "cash_outflows"
			// Not quantity * price_per_unit: NUMERIC(19,4) rounds a sub-cent
			// asset to 0 and its cost vanishes.
			amount = trade.ValueAtBuy
			if trade.Asset.InvestmentType != models.InvestmentCrypto {
				amount = amount.Add(trade.Fee)
			}
		}

		if trade.Currency != accCurrency {
			rate, ok := rates[utils.NewTradeExchangeRateKey(trade)]
			if !ok {
				return fmt.Errorf("no exchange rate resolved for trade %d (%s to %s)", trade.ID, trade.Currency, accCurrency)
			}
			amount = amount.Mul(rate)
		}

		if err := s.accRepo.AddToDailyBalance(ctx, tx, trade.Asset.AccountID, txnDate, field, amount); err != nil {
			return err
		}

		if earliest, ok := earliestByAccount[trade.Asset.AccountID]; !ok || txnDate.Before(earliest) {
			earliestByAccount[trade.Asset.AccountID] = txnDate
		}
		accountCurrency[trade.Asset.AccountID] = accCurrency
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for accountID, earliestDate := range earliestByAccount {
		currency := accountCurrency[accountID]

		if err := s.accRepo.FrontfillBalances(ctx, tx, accountID, currency, earliestDate); err != nil {
			return err
		}

		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accountID, currency, earliestDate, today); err != nil {
			return err
		}
	}

	return nil
}

func (s *InvestmentService) rebuildSnapshots(ctx context.Context, tx *gorm.DB, userID int64) error {
	accounts, err := s.accRepo.FindAllAccounts(ctx, tx, userID, true, false)
	if err != nil {
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, acc := range accounts {
		earliest, err := s.accRepo.GetAccountOpeningAsOf(ctx, tx, acc.ID)
		if errors.Is(err, sql.ErrNoRows) {
			continue // no balance rows, nothing to rebuild from
		}
		if err != nil {
			return fmt.Errorf("failed to get opening date for account %d: %w", acc.ID, err)
		}

		if err := s.accRepo.FrontfillBalances(ctx, tx, acc.ID, acc.Currency, earliest); err != nil {
			return fmt.Errorf("failed to frontfill balances for account %d: %w", acc.ID, err)
		}

		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, acc.ID, acc.Currency, earliest, today); err != nil {
			return fmt.Errorf("failed to rebuild snapshots for account %d: %w", acc.ID, err)
		}
	}

	return nil
}

func (s *InvestmentService) correctTradeFeeAccounting(ctx context.Context, tx *gorm.DB, userID int64) error {
	trades, err := s.repo.FindAllTradesByUserID(ctx, tx, userID)
	if err != nil {
		return err
	}

	affectedAssetIDs := map[int64]bool{}

	for _, trade := range trades {
		if trade.TradeType != models.InvestmentBuy {
			continue
		}
		if trade.Asset.InvestmentType != models.InvestmentStock && trade.Asset.InvestmentType != models.InvestmentETF {
			continue
		}
		if !trade.Fee.IsPositive() {
			continue
		}

		corrected := trade.Quantity.Mul(trade.PricePerUnit)
		if err := s.repo.CorrectTradeValueAtBuy(ctx, tx, trade.ID, corrected); err != nil {
			return err
		}
		affectedAssetIDs[trade.AssetID] = true
	}

	for assetID := range affectedAssetIDs {
		if err := s.repo.RecalculateAssetFromTrades(ctx, tx, assetID, userID); err != nil {
			return err
		}
	}

	return nil
}

func (s *InvestmentService) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date *time.Time) (decimal.Decimal, error) {
	if fromCurrency == toCurrency {
		return decimal.NewFromFloat(1.0), nil
	}

	if s.priceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}

	// For historical rates, check the DB cache first
	if date != nil {
		cached, found, err := s.repo.GetCachedExchangeRate(ctx, nil, fromCurrency, toCurrency, *date)
		if err != nil {
			return decimal.Zero, err
		}
		if found {
			return cached, nil
		}

		rate, err := s.priceFetchClient.GetExchangeRateOnDate(ctx, fromCurrency, toCurrency, *date)
		if err != nil {
			return decimal.Zero, err
		}

		result := decimal.NewFromFloat(rate)
		if upsertErr := s.repo.UpsertExchangeRate(ctx, nil, models.ExchangeRateHistory{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			AsOf:         *date,
			Rate:         result,
		}); upsertErr != nil {
			s.logger.Warn("Failed to cache exchange rate", zap.Error(upsertErr))
		}

		return result, nil
	}

	// Live rate — never cache
	rate, err := s.priceFetchClient.GetExchangeRate(ctx, fromCurrency, toCurrency)
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
		// Stock/ETF: pure trade value; fee is accounted for separately in cash outflows
		valueAtBuy = req.Quantity.Mul(req.PricePerUnit)
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
	utils.CompareChanges(exHold.Name, hold.Name, changes, "name")

	if changes.HasChanges() {
		changes.Stamp("id", strconv.FormatInt(holdID, 10))
		changes.Stamp("asset", exHold.Ticker)
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       "update",
			Category:    "investment_asset",
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

	utils.CompareChanges(oldDesc, newDesc, changes, "description")

	if changes.HasChanges() {
		changes.Stamp("id", strconv.FormatInt(txnID, 10))
		changes.Stamp("asset", asset.Ticker)
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "delete",
		Category:    "investment_asset",
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

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *InvestmentService) RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error {
	if s.priceFetchClient != nil {
		asset, err := s.repo.FindInvestmentAssetByID(ctx, nil, assetID, userID)
		if err != nil {
			return err
		}
		priceData, err := s.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err == nil && priceData != nil && priceData.Price > 0 {
			now := time.Now().UTC()
			price := decimal.NewFromFloat(priceData.Price)
			if err := s.repo.UpdateAssetCurrentPrice(ctx, nil, assetID, price, now); err != nil {
				return err
			}
			if err := s.repo.UpdateTradesPnLForAsset(ctx, nil, assetID, price, asset.InvestmentType, now); err != nil {
				return err
			}
			today := now.Truncate(24 * time.Hour)
			if err := s.repo.UpsertAssetPrice(ctx, nil, []models.AssetPriceHistory{
				{AssetID: assetID, AsOf: today, Price: price, Currency: priceData.Currency},
			}); err != nil {
				return err
			}
		}
	}
	return s.repo.RecalculateAssetFromTrades(ctx, nil, assetID, userID)
}

func (s *InvestmentService) GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error) {
	return s.repo.GetAssetIDsForAccount(ctx, nil, accountID, userID)
}

func (s *InvestmentService) GetUserIDsWithInvestments(ctx context.Context) ([]int64, error) {
	return s.repo.GetUserIDsWithInvestments(ctx, nil)
}

func (s *InvestmentService) GetAssetsForPriceBackfill(ctx context.Context) ([]models.AssetBackfillRow, error) {
	return s.repo.FindAssetsForPriceBackfill(ctx, nil)
}

func (s *InvestmentService) GetTickersForPriceSync(ctx context.Context) ([]models.AssetPriceSyncRow, error) {
	return s.repo.FindTickersForPriceSync(ctx, nil)
}

func (s *InvestmentService) ApplyTickerPrice(ctx context.Context, ticker string, price decimal.Decimal, currency string, now time.Time) ([]models.AssetPriceChange, error) {
	if price.IsZero() || price.IsNegative() {
		return nil, fmt.Errorf("invalid price for ticker %s: %s", ticker, price.String())
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	assets, err := s.repo.FindActiveAssetsByTicker(ctx, tx, ticker)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	today := now.UTC().Truncate(24 * time.Hour)
	changes := make([]models.AssetPriceChange, 0, len(assets))

	for _, asset := range assets {
		if utils.IsExtremePriceDrop(asset.CurrentPrice, price) {
			s.logger.Warn("Extreme price drop detected — skipping update to prevent data corruption",
				zap.Int64("asset_id", asset.ID),
				zap.String("ticker", asset.Ticker),
				zap.String("old_price", asset.CurrentPrice.String()),
				zap.String("new_price", price.String()))
			continue
		}

		currentValue := asset.Quantity.Mul(price)
		costBasis := asset.ValueAtBuy
		if asset.InvestmentType != models.InvestmentCrypto {
			costBasis = asset.ValueAtBuy.Add(asset.TotalFees)
		}
		profitLoss := currentValue.Sub(costBasis)
		var profitLossPercent decimal.Decimal
		if !costBasis.IsZero() {
			profitLossPercent = profitLoss.Div(costBasis)
		}

		if err := s.repo.UpdateAssetPriceAndValue(ctx, tx, asset.ID, price, currentValue, profitLoss, profitLossPercent, now); err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := s.repo.UpdateTradesPnLForAsset(ctx, tx, asset.ID, price, asset.InvestmentType, now); err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := s.repo.UpsertAssetPrice(ctx, tx, []models.AssetPriceHistory{
			{AssetID: asset.ID, AsOf: today, Price: price, Currency: currency},
		}); err != nil {
			s.logger.Warn("Failed to upsert asset price history",
				zap.Int64("asset_id", asset.ID),
				zap.Error(err))
		}

		changes = append(changes, models.AssetPriceChange{
			AssetID:  asset.ID,
			UserID:   asset.UserID,
			Ticker:   asset.Ticker,
			OldPrice: asset.CurrentPrice,
			NewPrice: price,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return changes, nil
}

func (s *InvestmentService) GetActiveCurrencyPairs(ctx context.Context) ([]models.CurrencyPair, error) {
	return s.repo.FindActiveCurrencyPairs(ctx, nil)
}

func (s *InvestmentService) GetUserIDsWithActiveInvestments(ctx context.Context) ([]int64, error) {
	return s.repo.FindUserIDsWithActiveInvestments(ctx, nil)
}

func (s *InvestmentService) UpdateSnapshotMarketValues(ctx context.Context, userID int64) error {
	return s.accRepo.UpdateSnapshotMarketValues(ctx, nil, userID, nil)
}

func (s *InvestmentService) FetchInvestmentIncomeByAsset(ctx context.Context, userID int64, assetID int64, p utils.PaginationParams) ([]models.InvestmentIncome, *utils.Paginator, error) {
	totalRecords, err := s.repo.CountInvestmentIncome(ctx, nil, assetID, userID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage
	records, err := s.repo.GetInvestmentIncomeByAsset(ctx, nil, assetID, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder)
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

func (s *InvestmentService) CreateInvestmentIncome(ctx context.Context, userID int64, req *models.InvestmentIncomeReq) (int64, error) {
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
		return 0, fmt.Errorf("can't find asset: %w", err)
	}

	if req.IncomeType == models.IncomeTypeStaking {
		if req.Quantity == nil || !req.Quantity.IsPositive() {
			tx.Rollback()
			return 0, fmt.Errorf("quantity is required and must be positive for staking rewards")
		}
	} else {
		if req.Amount == nil || !req.Amount.IsPositive() {
			tx.Rollback()
			return 0, fmt.Errorf("amount is required and must be positive for dividend income")
		}
	}

	// Staking rewards carry their value on receipt as cost basis, so a missing price is fatal
	var incomeAmount decimal.Decimal
	if req.IncomeType == models.IncomeTypeStaking {
		priceData, err := s.priceFetchClient.GetAssetPriceOnDate(ctx, asset.Ticker, asset.InvestmentType, req.TxnDate)
		if err != nil || priceData == nil || priceData.Price <= 0 {
			tx.Rollback()
			s.logger.Warn("Failed to price staking reward",
				zap.String("ticker", asset.Ticker),
				zap.Time("txn_date", req.TxnDate),
				zap.Error(err))
			return 0, fmt.Errorf("could not fetch a price for %s on %s, try again later",
				asset.Ticker, req.TxnDate.Format("2006-01-02"))
		}
		incomeAmount = req.Quantity.Mul(decimal.NewFromFloat(priceData.Price))
	} else {
		incomeAmount = *req.Amount
	}

	record := models.InvestmentIncome{
		UserID:      userID,
		AssetID:     asset.ID,
		TxnDate:     req.TxnDate.UTC().Truncate(24 * time.Hour),
		IncomeType:  req.IncomeType,
		Quantity:    req.Quantity,
		Amount:      incomeAmount,
		TaxWithheld: req.TaxWithheld,
		Currency:    req.Currency,
		Notes:       req.Notes,
	}

	id, err := s.repo.CreateInvestmentIncome(ctx, tx, &record)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if req.IncomeType == models.IncomeTypeStaking {
		if err := s.repo.RecalculateAssetFromTrades(ctx, tx, asset.ID, userID); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if req.IncomeType == models.IncomeTypeDividend {
		category, err := s.txnRepo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to find uncategorized category: %w", err)
		}
		grossAmount := *req.Amount
		if req.TaxWithheld != nil {
			grossAmount = grossAmount.Sub(*req.TaxWithheld)
		}
		dividendAmount := grossAmount
		if req.Currency != asset.Account.Currency {
			rate, err := s.GetExchangeRate(ctx, req.Currency, asset.Account.Currency, &req.TxnDate)
			if err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("failed to get exchange rate for dividend: %w", err)
			}
			dividendAmount = grossAmount.Mul(rate)
		}
		desc := "Dividend: " + asset.Ticker
		txn := &models.Transaction{
			UserID:          userID,
			AccountID:       asset.AccountID,
			CategoryID:      &category.ID,
			TransactionType: "income",
			Amount:          dividendAmount,
			Currency:        asset.Account.Currency,
			TxnDate:         record.TxnDate,
			Description:     &desc,
			IsSystem:        true,
		}
		txnID, err := s.txnRepo.InsertTransaction(ctx, tx, txn)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to create linked dividend transaction: %w", err)
		}
		if err := tx.Model(&models.InvestmentIncome{}).Where("id = ?", id).Update("linked_transaction_id", txnID).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to link dividend transaction: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	if req.IncomeType == models.IncomeTypeStaking {
		txnDate := req.TxnDate.UTC().Truncate(24 * time.Hour)
		if err := s.jobDispatcher.Dispatch(ctx, jobqueue.SyncAssetAfterTradeArgs{
			UserID:         userID,
			AssetID:        asset.ID,
			Ticker:         asset.Ticker,
			InvestmentType: asset.InvestmentType,
			TradeDate:      txnDate,
		}); err != nil {
			s.logger.Warn("Failed to dispatch post-income sync job", zap.Error(err))
		}
	}

	return id, nil
}

func (s *InvestmentService) DeleteInvestmentIncome(ctx context.Context, userID int64, id int64) error {
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

	income, err := s.repo.FindInvestmentIncomeByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find investment income record: %w", err)
	}

	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, income.AssetID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find asset: %w", err)
	}

	if income.LinkedTransactionID != nil {
		if err := s.txnRepo.DeleteTransaction(ctx, tx, *income.LinkedTransactionID, userID); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete linked dividend transaction: %w", err)
		}
	}

	if err := s.repo.DeleteInvestmentIncome(ctx, tx, id, userID); err != nil {
		tx.Rollback()
		return err
	}

	if income.IncomeType == models.IncomeTypeStaking {
		if err := s.repo.RecalculateAssetFromTrades(ctx, tx, asset.ID, userID); err != nil {
			tx.Rollback()
			return err
		}

		incomeDate := income.TxnDate.UTC().Truncate(24 * time.Hour)
		today := time.Now().UTC().Truncate(24 * time.Hour)

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, incomeDate, asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

		if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, incomeDate); err != nil {
			tx.Rollback()
			return err
		}

		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, asset.AccountID, asset.Account.Currency, incomeDate, today); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *InvestmentService) BackfillAssetPriceHistory(ctx context.Context, assetID int64, ticker string, from, to time.Time) error {
	if s.priceFetchClient == nil {
		return nil
	}

	from = from.UTC().Truncate(24 * time.Hour)
	to = to.UTC().Truncate(24 * time.Hour)
	if from.After(to) {
		return nil
	}

	existing, err := s.repo.GetPriceHistoryForAsset(ctx, nil, assetID)
	if err != nil {
		return err
	}

	existingSet := make(map[string]bool, len(existing))
	for _, p := range existing {
		existingSet[p.AsOf.UTC().Truncate(24*time.Hour).Format(time.DateOnly)] = true
	}

	prices, err := s.priceFetchClient.GetAssetPriceRange(ctx, ticker, from, to)
	if err != nil {
		return fmt.Errorf("asset %d (%s): %w", assetID, ticker, err)
	}

	var batch []models.AssetPriceHistory
	written := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := s.repo.UpsertAssetPrice(ctx, nil, batch); err != nil {
			return err
		}
		written += len(batch)
		batch = nil
		return nil
	}

	for _, p := range prices {
		if existingSet[p.Date.Format(time.DateOnly)] {
			continue
		}

		price := decimal.NewFromFloat(p.Price)
		if price.IsZero() || price.IsNegative() {
			continue
		}

		batch = append(batch, models.AssetPriceHistory{
			AssetID:  assetID,
			AsOf:     p.Date,
			Price:    price,
			Currency: p.Currency,
		})

		if len(batch) >= priceBackfillFlushSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if err := flush(); err != nil {
		return err
	}

	s.logger.Debug("Price history backfilled",
		zap.String("ticker", ticker),
		zap.Int("fetched", len(prices)),
		zap.Int("written", written))

	return nil
}

func (s *InvestmentService) FetchTaxBrackets(ctx context.Context, userID int64) ([]models.InvestmentTaxBracket, error) {
	return s.repo.FindTaxBracketsByUser(ctx, nil, userID)
}

func (s *InvestmentService) InsertTaxBracket(ctx context.Context, userID int64, req *models.InvestmentTaxBracketReq) (int64, error) {
	existing, err := s.repo.FindTaxBracketsByUserAndType(ctx, nil, userID, req.InvestmentType)
	if err != nil {
		return 0, err
	}
	for _, b := range existing {
		newCoversExisting := req.ToDays == nil || *req.ToDays >= b.MinDaysHeld
		existingCoversNew := b.ToDays == nil || *b.ToDays >= req.MinDaysHeld
		if newCoversExisting && existingCoversNew {
			toDays := "∞"
			if b.ToDays != nil {
				toDays = strconv.Itoa(*b.ToDays)
			}
			return 0, fmt.Errorf("bracket overlaps with existing bracket (days %d–%s)", b.MinDaysHeld, toDays)
		}
	}
	record := models.InvestmentTaxBracket{
		UserID:         userID,
		InvestmentType: req.InvestmentType,
		MinDaysHeld:    req.MinDaysHeld,
		ToDays:         req.ToDays,
		TaxablePercent: req.TaxablePercent,
		Label:          req.Label,
	}
	return s.repo.InsertTaxBracket(ctx, nil, &record)
}

func (s *InvestmentService) UpdateTaxBracket(ctx context.Context, userID int64, id int64, req *models.InvestmentTaxBracketReq) error {
	return s.repo.UpdateTaxBracket(ctx, nil, models.InvestmentTaxBracket{
		ID:             id,
		UserID:         userID,
		TaxablePercent: req.TaxablePercent,
		ToDays:         req.ToDays,
		Label:          req.Label,
	})
}

func (s *InvestmentService) DeleteTaxBracket(ctx context.Context, userID int64, id int64) error {
	return s.repo.DeleteTaxBracket(ctx, nil, id, userID)
}

func (s *InvestmentService) FetchTaxSettings(ctx context.Context, userID int64) (models.InvestmentTaxSettings, error) {
	return s.repo.FindTaxSettings(ctx, nil, userID)
}

func (s *InvestmentService) SaveTaxSettings(ctx context.Context, userID int64, req *models.InvestmentTaxSettingsReq) error {
	return s.repo.UpsertTaxSettings(ctx, nil, models.InvestmentTaxSettings{
		UserID:                userID,
		LossOffsettingEnabled: req.LossOffsettingEnabled,
	})
}

func (s *InvestmentService) CopyTaxBrackets(ctx context.Context, userID int64, fromType, toType models.InvestmentType) error {
	target, err := s.repo.FindTaxBracketsByUserAndType(ctx, nil, userID, toType)
	if err != nil {
		return err
	}
	if len(target) > 0 {
		return fmt.Errorf("%s already has brackets configured", toType)
	}

	source, err := s.repo.FindTaxBracketsByUserAndType(ctx, nil, userID, fromType)
	if err != nil {
		return err
	}
	if len(source) == 0 {
		return fmt.Errorf("%s has no brackets to copy", fromType)
	}

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

	for _, b := range source {
		record := models.InvestmentTaxBracket{
			UserID:         userID,
			InvestmentType: toType,
			MinDaysHeld:    b.MinDaysHeld,
			ToDays:         b.ToDays,
			TaxablePercent: b.TaxablePercent,
			Label:          b.Label,
		}
		if _, err := s.repo.InsertTaxBracket(ctx, tx, &record); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *InvestmentService) MigrateZeroCostTradesForAsset(ctx context.Context, userID, assetID int64, trades []models.InvestmentTrade) error {
	if len(trades) == 0 {
		return nil
	}

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

	asset, err := s.repo.FindInvestmentAssetByID(ctx, tx, assetID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("asset %d not found: %w", assetID, err)
	}

	incomeType := models.IncomeTypeStaking
	if asset.InvestmentType != models.InvestmentCrypto {
		incomeType = models.IncomeTypeDividend
	}

	var uncategorizedCatID *int64
	if incomeType == models.IncomeTypeDividend {
		cat, err := s.txnRepo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to find uncategorized category: %w", err)
		}
		uncategorizedCatID = &cat.ID
	}

	earliestDate := trades[0].TxnDate.UTC().Truncate(24 * time.Hour)

	for _, trade := range trades {
		txnDate := trade.TxnDate.UTC().Truncate(24 * time.Hour)
		if txnDate.Before(earliestDate) {
			earliestDate = txnDate
		}

		var amount decimal.Decimal
		priceData, err := s.priceFetchClient.GetAssetPriceOnDate(ctx, asset.Ticker, asset.InvestmentType, txnDate)
		if err == nil && priceData != nil && priceData.Price > 0 {
			amount = trade.Quantity.Mul(decimal.NewFromFloat(priceData.Price))
		}

		record := models.InvestmentIncome{
			UserID:     userID,
			AssetID:    assetID,
			TxnDate:    txnDate,
			IncomeType: incomeType,
			Currency:   trade.Currency,
			Amount:     amount,
		}
		if incomeType == models.IncomeTypeStaking {
			record.Quantity = &trade.Quantity
		}

		incomeID, err := s.repo.CreateInvestmentIncome(ctx, tx, &record)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create income record for trade %d: %w", trade.ID, err)
		}

		if incomeType == models.IncomeTypeDividend && amount.IsPositive() {
			dividendAmount := amount
			if trade.Currency != asset.Account.Currency {
				rate, err := s.GetExchangeRate(ctx, trade.Currency, asset.Account.Currency, &txnDate)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("exchange rate error for trade %d: %w", trade.ID, err)
				}
				dividendAmount = amount.Mul(rate)
			}
			desc := "Dividend: " + asset.Ticker
			txn := &models.Transaction{
				UserID:          userID,
				AccountID:       asset.AccountID,
				CategoryID:      uncategorizedCatID,
				TransactionType: "income",
				Amount:          dividendAmount,
				Currency:        asset.Account.Currency,
				TxnDate:         txnDate,
				Description:     &desc,
				IsSystem:        true,
			}
			txnID, err := s.txnRepo.InsertTransaction(ctx, tx, txn)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create dividend transaction for trade %d: %w", trade.ID, err)
			}
			if err := tx.Model(&models.InvestmentIncome{}).Where("id = ?", incomeID).Update("linked_transaction_id", txnID).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to link dividend transaction for trade %d: %w", trade.ID, err)
			}
		}

		if err := s.repo.DeleteInvestmentTrade(ctx, tx, trade.ID); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete trade %d: %w", trade.ID, err)
		}
	}

	if err := s.repo.RecalculateAssetFromTrades(ctx, tx, assetID, userID); err != nil {
		tx.Rollback()
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, earliestDate, asset.Account.Currency); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, earliestDate); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, asset.AccountID, asset.Account.Currency, earliestDate, today); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InvestmentService) FetchPortfolioAllocation(ctx context.Context, userID int64, currency string) (*models.PortfolioAllocation, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
		if err != nil {
			return nil, err
		}
		currency = settings.DefaultCurrency
	}

	rows, err := s.repo.FetchPortfolioAllocation(ctx, nil, userID, currency)
	if err != nil {
		return nil, err
	}

	byType := make(map[string]*models.AllocationRow)
	byTicker := make(map[string]*models.AllocationRow)
	byCurrency := make(map[string]*models.AllocationRow)
	byAccount := make(map[string]*models.AllocationRow)

	total := decimal.Zero
	unpriced := 0

	for _, row := range rows {
		if !row.Priced {
			unpriced++
			continue
		}

		value, err := decimal.NewFromString(row.ValueText)
		if err != nil {
			return nil, fmt.Errorf("invalid allocation value for %s: %w", row.Ticker, err)
		}
		if !value.IsPositive() {
			continue
		}

		var investmentTypeLabels = map[models.InvestmentType]string{
			models.InvestmentStock:  "Stock",
			models.InvestmentETF:    "ETF",
			models.InvestmentCrypto: "Crypto",
		}

		label, ok := investmentTypeLabels[row.InvestmentType]
		if !ok {
			label = string(row.InvestmentType)
		}

		total = total.Add(value)
		utils.AddAllocation(byType, string(row.InvestmentType), label, value)
		utils.AddAllocation(byTicker, row.Ticker, row.Name, value)
		utils.AddAllocation(byCurrency, row.SourceCurrency, row.SourceCurrency, value)
		utils.AddAllocation(byAccount, strconv.FormatInt(row.AccountID, 10), row.AccountName, value)
	}

	return &models.PortfolioAllocation{
		Currency:       currency,
		TotalValue:     total,
		UnpricedAssets: unpriced,
		Groups: map[string][]models.AllocationRow{
			"type":     utils.AllocationRows(byType, total),
			"ticker":   utils.AllocationRows(byTicker, total),
			"currency": utils.AllocationRows(byCurrency, total),
			"account":  utils.AllocationRows(byAccount, total),
		},
	}, nil
}
