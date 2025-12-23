package services

import (
	"context"
	"fmt"
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
	FetchInvestmentHoldingsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentHolding, *utils.Paginator, error)
	FetchAllInvestmentHoldings(ctx context.Context, userID int64) ([]models.InvestmentHolding, error)
	FetchInvestmentHoldingByID(ctx context.Context, userID int64, id int64) (*models.InvestmentHolding, error)
	FetchInvestmentTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTransaction, *utils.Paginator, error)
	FetchInvestmentTransactionByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTransaction, error)
	InsertHolding(ctx context.Context, userID int64, req *models.InvestmentHoldingReq) (int64, error)
	InsertInvestmentTransaction(ctx context.Context, userID int64, req *models.InvestmentTransactionReq) (int64, error)
	UpdateInvestmentAccountBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64, asOf time.Time, currency string) error
	UpdateInvestmentAccountBalanceRange(ctx context.Context, tx *gorm.DB, accountID, userID int64, fromDate, toDate time.Time, currency string) error
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

func (s *InvestmentService) FetchInvestmentHoldingsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentHolding, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentHoldings(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentHoldings(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
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

func (s *InvestmentService) FetchAllInvestmentHoldings(ctx context.Context, userID int64) ([]models.InvestmentHolding, error) {
	return s.repo.FindAllInvestmentHoldings(ctx, nil, userID)
}

func (s *InvestmentService) FetchInvestmentHoldingByID(ctx context.Context, userID int64, id int64) (*models.InvestmentHolding, error) {

	record, err := s.repo.FindInvestmentHoldingByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) FetchInvestmentTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTransaction, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentTransactions(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentTransactions(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
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

func (s *InvestmentService) FetchInvestmentTransactionByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTransaction, error) {

	record, err := s.repo.FindInvestmentTransactionByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) InsertHolding(ctx context.Context, userID int64, req *models.InvestmentHoldingReq) (int64, error) {
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
	var ut string

	if s.priceFetchClient != nil {
		var ticker, exchangeOrCurrency string
		ut = strings.ToUpper(req.Ticker)

		switch req.InvestmentType {
		case models.InvestmentCrypto:
			// "BTC-USD"
			if parts := strings.Split(ut, "-"); len(parts) == 2 {
				ticker = parts[0]
				exchangeOrCurrency = parts[1]
			} else {
				ticker = ut
			}
		case models.InvestmentStock, models.InvestmentETF:
			// "IWDA.L"
			if parts := strings.Split(ut, "."); len(parts) == 2 {
				ticker = parts[0]
				exchangeOrCurrency = parts[1]
			} else {
				ticker = ut
			}
		}

		// Fetch price
		priceData, err := s.priceFetchClient.GetAssetPrice(ctx, ticker, req.InvestmentType, exchangeOrCurrency)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to fetch price for ticker '%s': %w", ticker, err)
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

	hold := models.InvestmentHolding{
		UserID:          userID,
		AccountID:       account.ID,
		InvestmentType:  req.InvestmentType,
		Name:            req.Name,
		Ticker:          ut,
		Quantity:        req.Quantity,
		AverageBuyPrice: decimal.Zero,
		CurrentPrice:    currentPrice,
		LastPriceUpdate: lastPriceUpdate,
	}

	holdID, err := s.repo.InsertHolding(ctx, tx, &hold)
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

	holdings, err := s.repo.FindHoldingsByAccountID(ctx, tx, accountID, userID)
	if err != nil {
		return err
	}

	if len(holdings) == 0 {
		return nil
	}

	var totalPnLAsOf decimal.Decimal

	for _, holding := range holdings {

		// Fetch the price for this specific date
		price, err := s.fetchPriceForHolding(ctx, holding, asOf)
		if err != nil {
			continue
		}
		// Calculate P&L for this holding at this date
		currentValueAtDate := holding.Quantity.Mul(price)
		pnlAtDate := currentValueAtDate.Sub(holding.ValueAtBuy)

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

func (s *InvestmentService) fetchPriceForHolding(ctx context.Context, holding models.InvestmentHolding, asOf time.Time) (decimal.Decimal, error) {
	if s.priceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}

	var ticker, exchangeOrCurrency string
	ut := strings.ToUpper(holding.Ticker)

	switch holding.InvestmentType {
	case models.InvestmentCrypto:
		if parts := strings.Split(ut, "-"); len(parts) == 2 {
			ticker = parts[0]
			exchangeOrCurrency = parts[1]
		} else {
			ticker = ut
			exchangeOrCurrency = "USD"
		}

	case models.InvestmentStock, models.InvestmentETF:
		if parts := strings.Split(ut, "."); len(parts) == 2 {
			ticker = parts[0]
			exchangeOrCurrency = parts[1]
		} else {
			ticker = ut
			exchangeOrCurrency = "AS"
		}
	}

	priceData, err := s.priceFetchClient.GetAssetPriceOnDate(ctx, ticker, holding.InvestmentType, asOf, exchangeOrCurrency)
	if err != nil {
		return decimal.Zero, err
	}

	return decimal.NewFromFloat(priceData.Price), nil
}

func (s *InvestmentService) generateCheckpointDates(fromDate, toDate time.Time) []time.Time {
	var dates []time.Time

	// Always include the transaction date (adjusted to next weekday if needed)
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

func (s *InvestmentService) InsertInvestmentTransaction(ctx context.Context, userID int64, req *models.InvestmentTransactionReq) (int64, error) {
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

	holding, err := s.repo.FindInvestmentHoldingByID(ctx, tx, req.HoldingID, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find holding with given id %w", err)
	}

	if req.TransactionType == models.InvestmentSell && req.Quantity.GreaterThan(holding.Quantity) {
		tx.Rollback()
		return 0, fmt.Errorf("cannot sell %s: insufficient quantity (have %s, trying to sell %s)",
			holding.Ticker,
			holding.Quantity.String(),
			req.Quantity.String())
	}

	// Get exchange rate
	exchangeRate := s.getExchangeRate(ctx, req.Currency)

	// Calculate fee
	fee := decimal.NewFromFloat(0.00)
	if req.Fee != nil {
		fee = *req.Fee
	}

	// Calculate effective quantity and value at buy
	effectiveQuantity, valueAtBuy := s.calculateTransactionValue(req, holding.InvestmentType, fee)

	// Fetch current price
	currentPrice, lastPriceUpdate := s.fetchCurrentPrice(ctx, holding)

	// Calculate transaction PnL
	txnCurrentValue, txnProfitLoss, txnProfitLossPercent := s.calculateTransactionPnL(
		req.Quantity,
		currentPrice,
		valueAtBuy,
	)

	txn := models.InvestmentTransaction{
		UserID:            userID,
		HoldingID:         req.HoldingID,
		TxnDate:           req.TxnDate,
		TransactionType:   req.TransactionType,
		Quantity:          effectiveQuantity,
		PricePerUnit:      req.PricePerUnit,
		Fee:               fee,
		ValueAtBuy:        valueAtBuy,
		CurrentValue:      txnCurrentValue,
		ProfitLoss:        txnProfitLoss,
		ProfitLossPercent: txnProfitLossPercent,
		Currency:          req.Currency,
		ExchangeRateToUSD: exchangeRate,
		Description:       req.Description,
	}

	txnID, err := s.repo.InsertInvestmentTransaction(ctx, tx, &txn)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Handle selling - calculate and record realized P&L
	if req.TransactionType == models.InvestmentSell {
		if err := s.handleSellTransaction(ctx, tx, holding, effectiveQuantity, req.PricePerUnit, req.TxnDate); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// Update holding with new quantity and price
	err = s.repo.UpdateHoldingAfterTransaction(
		ctx, tx, holding.ID, effectiveQuantity, req.PricePerUnit,
		currentPrice, lastPriceUpdate, req.TransactionType, valueAtBuy,
	)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Update unrealized P&L for remaining holdings
	if err := s.updateUnrealizedPnL(ctx, tx, holding, req.TxnDate); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", holding.Ticker, changes, "holding")
	utils.CompareChanges("", txn.Quantity.StringFixed(2), changes, "quantity")
	utils.CompareChanges("", txn.PricePerUnit.StringFixed(2), changes, "price_per_unit")
	utils.CompareChanges("", txn.Fee.StringFixed(2), changes, "fee")
	utils.CompareChanges("", txn.TxnDate.UTC().Format(time.RFC3339), changes, "date")
	utils.CompareChanges("", string(txn.TransactionType), changes, "type")
	utils.CompareChanges("", txn.Currency, changes, "currency")

	if txn.Description != nil {
		utils.CompareChanges("", *txn.Description, changes, "description")
	}

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "investment_transaction",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return txnID, nil
}

func (s *InvestmentService) handleSellTransaction(ctx context.Context, tx *gorm.DB, holding models.InvestmentHolding, quantitySold, salePrice decimal.Decimal, txnDate time.Time) error {

	proceeds := quantitySold.Mul(salePrice)
	costBasis := holding.AverageBuyPrice.Mul(quantitySold)
	realizedPnL := proceeds.Sub(costBasis)

	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, holding.AccountID, txnDate, holding.Account.Currency); err != nil {
		return err
	}

	if realizedPnL.GreaterThanOrEqual(decimal.Zero) {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, holding.AccountID, txnDate, "cash_inflows", realizedPnL); err != nil {
			return err
		}
	} else {
		if err := s.accRepo.AddToDailyBalance(ctx, tx, holding.AccountID, txnDate, "cash_outflows", realizedPnL.Abs()); err != nil {
			return err
		}
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, holding.AccountID, holding.Account.Currency, txnDate); err != nil {
		return err
	}

	return s.accRepo.UpsertSnapshotsFromBalances(
		ctx, tx,
		holding.UserID,
		holding.AccountID,
		holding.Account.Currency,
		txnDate.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	)
}

func (s *InvestmentService) updateUnrealizedPnL(ctx context.Context, tx *gorm.DB, holding models.InvestmentHolding, txnDate time.Time) error {
	// Update the checkpoint for transaction date
	if err := s.UpdateInvestmentAccountBalance(ctx, tx, holding.AccountID, holding.UserID, txnDate, holding.Account.Currency); err != nil {
		return err
	}

	// If transaction is in the past, update all checkpoints from txn_date to today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	txnDateNorm := txnDate.UTC().Truncate(24 * time.Hour)

	if txnDateNorm.Before(today) {
		return s.UpdateInvestmentAccountBalanceRange(ctx, tx, holding.AccountID, holding.UserID, txnDateNorm.AddDate(0, 0, 1), today, holding.Account.Currency)
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

func (s *InvestmentService) calculateTransactionValue(req *models.InvestmentTransactionReq, investmentType models.InvestmentType, fee decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
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

func (s *InvestmentService) fetchCurrentPrice(ctx context.Context, holding models.InvestmentHolding) (*decimal.Decimal, *time.Time) {
	if s.priceFetchClient == nil {
		return nil, nil
	}

	var ticker, exchangeOrCurrency string
	ut := strings.ToUpper(holding.Ticker)

	switch holding.InvestmentType {
	case models.InvestmentCrypto:
		if parts := strings.Split(ut, "-"); len(parts) == 2 {
			ticker = parts[0]
			exchangeOrCurrency = parts[1]
		} else {
			ticker = ut
		}
	case models.InvestmentStock, models.InvestmentETF:
		if parts := strings.Split(ut, "."); len(parts) == 2 {
			ticker = parts[0]
			exchangeOrCurrency = parts[1]
		} else {
			ticker = ut
		}
	}

	priceData, err := s.priceFetchClient.GetAssetPrice(ctx, ticker, holding.InvestmentType, exchangeOrCurrency)
	if err != nil {
		return nil, nil
	}

	price := decimal.NewFromFloat(priceData.Price)
	now := time.Unix(priceData.LastUpdate, 0)
	return &price, &now
}

func (s *InvestmentService) calculateTransactionPnL(quantity decimal.Decimal, currentPrice *decimal.Decimal, valueAtBuy decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	if currentPrice != nil && !currentPrice.IsZero() {
		currentValue := quantity.Mul(*currentPrice)
		profitLoss := currentValue.Sub(valueAtBuy)

		var profitLossPercent decimal.Decimal
		if !valueAtBuy.IsZero() {
			profitLossPercent = profitLoss.Div(valueAtBuy).Mul(decimal.NewFromInt(100))
		}

		return currentValue, profitLoss, profitLossPercent
	}

	return decimal.Zero, decimal.Zero, decimal.Zero
}
