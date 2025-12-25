package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/prices"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InvestmentServiceInterface interface {
	FetchInvestmentAssetsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentAsset, *utils.Paginator, error)
	FetchAllInvestmentAssets(ctx context.Context, userID int64) ([]models.InvestmentAsset, error)
	FetchInvestmentAssetByID(ctx context.Context, userID int64, id int64) (*models.InvestmentAsset, error)
	FetchInvestmentTradesPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTrade, *utils.Paginator, error)
	FetchInvestmentTradeByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTrade, error)
	InsertAsset(ctx context.Context, userID int64, req *models.InvestmentAssetReq) (int64, error)
	InsertInvestmentTrade(ctx context.Context, userID int64, req *models.InvestmentTradeReq) (int64, error)
	UpdateInvestmentAccountBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64, asOf time.Time, currency string) error
	UpdateInvestmentAccountBalanceRange(ctx context.Context, tx *gorm.DB, accountID, userID int64, fromDate, toDate time.Time, currency string) error
	UpdateInvestmentAsset(ctx context.Context, userID int64, id int64, req *models.InvestmentAssetReq) (int64, error)
	UpdateInvestmentTrade(ctx context.Context, userID int64, id int64, req *models.InvestmentTradeReq) (int64, error)
	DeleteInvestmentAsset(ctx context.Context, userID int64, id int64) error
	DeleteInvestmentTrade(ctx context.Context, userID int64, id int64) error
}

type InvestmentService struct {
	repo             repositories.InvestmentRepositoryInterface
	accRepo          repositories.AccountRepositoryInterface
	settingsRepo     *repositories.SettingsRepository
	loggingRepo      repositories.LoggingRepositoryInterface
	jobDispatcher    jobqueue.JobDispatcher
	priceFetchClient prices.PriceFetcher
}

func NewInvestmentService(
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
	priceFetchClient prices.PriceFetcher,
) *InvestmentService {
	return &InvestmentService{
		repo:             repo,
		accRepo:          accRepo,
		settingsRepo:     settingsRepo,
		jobDispatcher:    jobDispatcher,
		loggingRepo:      loggingRepo,
		priceFetchClient: priceFetchClient,
	}
}

var _ InvestmentServiceInterface = (*InvestmentService)(nil)

var stockTickerRegex = regexp.MustCompile(`^[A-Z]{1,6}(\.[A-Z]{1,5})?$`)

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

func (s *InvestmentService) FetchInvestmentTradesPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTrade, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentTrades(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentTrades(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
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
		AverageBuyPrice: decimal.Zero,
		CurrentPrice:    currentPrice,
		LastPriceUpdate: lastPriceUpdate,
	}

	holdID, err := s.repo.InsertAsset(ctx, tx, &hold)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	quantityString := hold.Quantity.StringFixed(2)

	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", hold.Name, changes, "name")
	utils.CompareChanges("", hold.Ticker, changes, "ticker")
	utils.CompareChanges("", string(hold.InvestmentType), changes, "type")
	utils.CompareChanges("", quantityString, changes, "quantity")

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

func (s *InvestmentService) UpdateInvestmentAccountBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64, asOf time.Time, currency string) error {

	assets, err := s.repo.FindAssetsByAccountID(ctx, tx, accountID, userID)
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		return nil
	}

	var totalPnLAsOf decimal.Decimal

	for _, asset := range assets {

		// Get historical quantity and spent for this asset as of this date
		qty, spent, err := s.repo.GetInvestmentTotalsUpToDate(ctx, tx, asset.ID, asOf)
		if err != nil || qty.IsZero() {
			continue
		}

		// Fetch the price for this specific date
		price, err := s.fetchPriceForDate(ctx, asset, asOf)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Calculate P&L using historical quantity and spent
		currentValueAtDate := qty.Mul(price)
		pnlAtDate := currentValueAtDate.Sub(spent)
		
		totalPnLAsOf = totalPnLAsOf.Add(pnlAtDate)
	}

	var previousPnL decimal.Decimal
	cumulativePnL, err := s.accRepo.GetCumulativeNonCashPnLBeforeDate(ctx, tx, accountID, asOf)

	if err == nil {
		previousPnL = cumulativePnL
	}

	// Calculate the delta (change in P&L since last checkpoint)
	delta := totalPnLAsOf.Sub(previousPnL)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, accountID, asOf, currency); err != nil {
		return err
	}

	// Clear existing non-cash flows for this date
	if err := s.accRepo.SetDailyBalance(ctx, tx, accountID, asOf, "non_cash_inflows", decimal.Zero); err != nil {
		return err
	}
	if err := s.accRepo.SetDailyBalance(ctx, tx, accountID, asOf, "non_cash_outflows", decimal.Zero); err != nil {
		return err
	}

	if delta.GreaterThan(decimal.Zero) {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, accountID, asOf, "non_cash_inflows", delta); err != nil {
			return err
		}
	} else if delta.LessThan(decimal.Zero) {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, accountID, asOf, "non_cash_outflows", delta.Abs()); err != nil {
			return err
		}
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, accountID, currency, asOf); err != nil {
		return err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(
		ctx,
		tx,
		userID,
		accountID,
		currency,
		asOf.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	); err != nil {
		return err
	}

	return nil
}

func (s *InvestmentService) UpdateInvestmentAccountBalanceRange(ctx context.Context, tx *gorm.DB, accountID, userID int64, fromDate, toDate time.Time, currency string) error {
	fromDate = fromDate.UTC().Truncate(24 * time.Hour)
	toDate = toDate.UTC().Truncate(24 * time.Hour)

	checkpoints := s.generateCheckpointDates(fromDate, toDate)
	for _, d := range checkpoints {
		if err := s.UpdateInvestmentAccountBalance(ctx, tx, accountID, userID, d, currency); err != nil {
			return err
		}
	}

	return nil
}

func (s *InvestmentService) fetchPriceForDate(ctx context.Context, asset models.InvestmentAsset, asOf time.Time) (decimal.Decimal, error) {
	if s.priceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}

	priceData, err := s.priceFetchClient.GetAssetPriceOnDate(ctx, asset.Ticker, asset.InvestmentType, asOf)
	if err != nil {
		return decimal.Zero, err
	}

	return decimal.NewFromFloat(priceData.Price), nil
}

func (s *InvestmentService) fetchCurrentPrice(ctx context.Context, asset models.InvestmentAsset) (*decimal.Decimal, *time.Time) {
	if s.priceFetchClient == nil {
		return nil, nil
	}

	priceData, err := s.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
	if err != nil {
		return nil, nil
	}

	price := decimal.NewFromFloat(priceData.Price)
	now := time.Unix(priceData.LastUpdate, 0)
	return &price, &now
}

func (s *InvestmentService) generateCheckpointDates(fromDate, toDate time.Time) []time.Time {
	var dates []time.Time

	// Always include the trade date (adjusted to next weekday if needed)
	dates = append(dates, utils.AdjustToWeekday(fromDate))

	current := fromDate.AddDate(0, 1, 0)
	current = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)

	for current.Before(toDate) {
		adjusted := utils.AdjustToWeekday(current)
		if len(dates) == 0 || !adjusted.Equal(dates[len(dates)-1]) {
			dates = append(dates, adjusted)
		}
		current = current.AddDate(0, 1, 0)
	}

	// Always include today (adjusted to weekday)
	adjustedToDate := utils.AdjustToWeekday(toDate)
	if len(dates) == 0 || !adjustedToDate.Equal(dates[len(dates)-1]) {
		dates = append(dates, adjustedToDate)
	}

	return dates
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

	// Validate amounts
	if req.TradeType == models.InvestmentSell && req.Quantity.GreaterThan(asset.Quantity) {
		tx.Rollback()
		return 0, fmt.Errorf("cannot sell %s: insufficient quantity (have %s, trying to sell %s)",
			asset.Ticker,
			asset.Quantity.String(),
			req.Quantity.String())
	}

	if req.TradeType == models.InvestmentBuy {

		availableBalance, err := s.accRepo.FindLatestBalance(ctx, tx, asset.AccountID, userID)
		if err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, fmt.Errorf("no balance record found for account")
			}
			return 0, fmt.Errorf("failed to get account balance: %w", err)
		}

		totalInvestmentValue, err := s.repo.FindTotalInvestmentValue(ctx, tx, asset.AccountID, userID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to calculate total investment value: %w", err)
		}

		purchaseCost := req.Quantity.Mul(req.PricePerUnit)
		if req.Fee != nil {
			purchaseCost = purchaseCost.Add(*req.Fee)
		}

		remainingBalance := availableBalance.EndBalance.Sub(totalInvestmentValue)
		if remainingBalance.LessThan(decimal.Zero) {
			tx.Rollback()
			return 0, fmt.Errorf("account balance inconsistency detected: total investments (%s) exceed account balance (%s)",
				totalInvestmentValue.StringFixed(2),
				availableBalance.EndBalance.StringFixed(2))
		}

		if purchaseCost.GreaterThan(remainingBalance) {
			tx.Rollback()
			return 0, fmt.Errorf("insufficient funds: need %s, but only %s available (balance: %s, invested: %s)",
				purchaseCost.StringFixed(2),
				remainingBalance.StringFixed(2),
				availableBalance.EndBalance.StringFixed(2),
				totalInvestmentValue.StringFixed(2))
		}
	}

	// Get exchange rate
	exchangeRate := s.getExchangeRate(ctx, req.Currency)

	// Calculate fee
	fee := decimal.NewFromFloat(0.00)
	if req.Fee != nil {
		fee = *req.Fee
	}

	// Calculate effective quantity and value at buy
	effectiveQuantity, valueAtBuy := s.calculateTradeValue(req, asset.InvestmentType, fee)

	// Fetch current price
	currentPrice, lastPriceUpdate := s.fetchCurrentPrice(ctx, asset)

	// Calculate trade PnL
	var txnCurrentValue, txnProfitLoss, txnProfitLossPercent, txnRealizedValue decimal.Decimal

	if req.TradeType == models.InvestmentSell {
		// For sells: calculate realized P&L using asset's average buy price
		txnRealizedValue = req.Quantity.Mul(req.PricePerUnit)
		costBasis := asset.AverageBuyPrice.Mul(req.Quantity)
		txnProfitLoss = txnRealizedValue.Sub(costBasis)

		// Current value still shows what the asset would be worth at market price
		txnCurrentValue, _, _ = s.calculateTradePnL(req.Quantity, currentPrice, costBasis)

		if !costBasis.IsZero() {
			txnProfitLossPercent = txnProfitLoss.Div(costBasis)
		}
	} else {
		// For buys: use current market price for unrealized P&L
		txnCurrentValue, txnProfitLoss, txnProfitLossPercent = s.calculateTradePnL(
			req.Quantity,
			currentPrice,
			valueAtBuy,
		)
	}

	var txnValueAtBuy decimal.Decimal
	if req.TradeType == models.InvestmentSell {
		txnValueAtBuy = asset.AverageBuyPrice.Mul(req.Quantity)
	} else {
		txnValueAtBuy = valueAtBuy
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
		ExchangeRateToUSD: exchangeRate,
		Description:       req.Description,
	}

	txnID, err := s.repo.InsertInvestmentTrade(ctx, tx, &txn)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Handle selling - calculate and record realized P&L
	if req.TradeType == models.InvestmentSell {
		if err := s.handleSellTrade(ctx, tx, asset, effectiveQuantity, req.PricePerUnit, req.TxnDate); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// Update asset with new quantity and price
	err = s.repo.UpdateAssetAfterTrade(
		ctx, tx, asset.ID, effectiveQuantity, req.PricePerUnit,
		currentPrice, lastPriceUpdate, req.TradeType, valueAtBuy,
	)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Update unrealized P&L for remaining assets
	if err := s.updateUnrealizedPnL(ctx, tx, asset, req.TxnDate); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
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

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

func (s *InvestmentService) handleSellTrade(ctx context.Context, tx *gorm.DB, asset models.InvestmentAsset, quantitySold, salePrice decimal.Decimal, txnDate time.Time) error {

	proceeds := quantitySold.Mul(salePrice)
	costBasis := asset.AverageBuyPrice.Mul(quantitySold)
	realizedPnL := proceeds.Sub(costBasis)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txnDate, asset.Account.Currency); err != nil {
		return err
	}

	if realizedPnL.GreaterThanOrEqual(decimal.Zero) {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_inflows", realizedPnL); err != nil {
			return err
		}
	} else {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txnDate, "cash_outflows", realizedPnL.Abs()); err != nil {
			return err
		}
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, asset.AccountID, asset.Account.Currency, txnDate); err != nil {
		return err
	}

	return s.accRepo.UpsertSnapshotsFromBalances(
		ctx, tx,
		asset.UserID,
		asset.AccountID,
		asset.Account.Currency,
		txnDate.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	)
}

func (s *InvestmentService) updateUnrealizedPnL(ctx context.Context, tx *gorm.DB, asset models.InvestmentAsset, txnDate time.Time) error {
	// Update the checkpoint for trade date
	if err := s.UpdateInvestmentAccountBalance(ctx, tx, asset.AccountID, asset.UserID, txnDate, asset.Account.Currency); err != nil {
		return err
	}

	// If trade is in the past, update all checkpoints from txn_date to today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	txnDateNorm := txnDate.UTC().Truncate(24 * time.Hour)

	if txnDateNorm.Before(today) {
		return s.UpdateInvestmentAccountBalanceRange(ctx, tx, asset.AccountID, asset.UserID, txnDateNorm.AddDate(0, 0, 1), today, asset.Account.Currency)
	}

	return nil
}

func (s *InvestmentService) getExchangeRate(ctx context.Context, currency string) decimal.Decimal {
	if s.priceFetchClient != nil {
		rate, err := s.priceFetchClient.GetExchangeRate(ctx, currency)
		if err == nil {
			return decimal.NewFromFloat(rate)
		}
	}
	return decimal.NewFromFloat(1.0)
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
	utils.CompareChanges("", exHold.Ticker, changes, "asset")
	utils.CompareChanges(exHold.Name, hold.Name, changes, "name")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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
		utils.CompareChanges("", asset.Ticker, changes, "asset")
		utils.CompareChanges(oldDesc, newDesc, changes, "description")
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

	// Get earliest trade date for this asset
	earliestTxnDate, err := s.repo.GetEarliestTradeDate(ctx, tx, id, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return err
	}

	// Find all sell trades to reverse their realized P&L
	sellTxns, err := s.repo.FindSellTradesByAssetID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Reverse realized P&L from all sells
	for _, txn := range sellTxns {
		if txn.RealizedValue == decimal.Zero {
			continue
		}

		costBasis := txn.ValueAtBuy
		realizedPnL := txn.RealizedValue.Sub(costBasis)

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, txn.TxnDate, asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

		if realizedPnL.GreaterThanOrEqual(decimal.Zero) {
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txn.TxnDate, "cash_inflows", realizedPnL.Neg()); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, txn.TxnDate, "cash_outflows", realizedPnL.Abs().Neg()); err != nil {
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

	// Clear non-cash flows from earliest trade date
	if !earliestTxnDate.IsZero() {
		if err := s.accRepo.ClearNonCashFlowsFromDate(ctx, tx, asset.AccountID, earliestTxnDate); err != nil {
			tx.Rollback()
			return err
		}

		// Recalculate unrealized P&L for remaining assets in this account
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if err := s.UpdateInvestmentAccountBalanceRange(ctx, tx, asset.AccountID, userID, earliestTxnDate, today, asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

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

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

	// Reverse sell realized P&L if it was a sell
	if exTxn.TradeType == models.InvestmentSell {
		costBasis := exTxn.ValueAtBuy
		realizedPnL := exTxn.RealizedValue.Sub(costBasis)

		if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, asset.AccountID, exTxn.TxnDate, asset.Account.Currency); err != nil {
			tx.Rollback()
			return err
		}

		if realizedPnL.GreaterThanOrEqual(decimal.Zero) {
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, exTxn.TxnDate, "cash_inflows", realizedPnL.Neg()); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := s.accRepo.AddToDailyBalance(ctx, tx, asset.AccountID, exTxn.TxnDate, "cash_outflows", realizedPnL.Abs().Neg()); err != nil {
				tx.Rollback()
				return err
			}
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

	// Clear non-cash flows from txn date forward
	if err := s.accRepo.ClearNonCashFlowsFromDate(ctx, tx, asset.AccountID, exTxn.TxnDate); err != nil {
		tx.Rollback()
		return err
	}

	// Recalculate unrealized P&L from txn date to today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	txnDateNorm := exTxn.TxnDate.UTC().Truncate(24 * time.Hour)

	if err := s.UpdateInvestmentAccountBalanceRange(ctx, tx, asset.AccountID, userID, txnDateNorm, today, asset.Account.Currency); err != nil {
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

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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
