package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionServiceInterface interface {
	FetchTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeDeleted bool, accountID *int64) ([]models.Transaction, *utils.Paginator, error)
	FetchTransfersPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeDeleted bool, accountID *int64) ([]models.Transfer, *utils.Paginator, error)
	FetchTransactionByID(ctx context.Context, userID int64, id int64, includeDeleted bool) (*models.Transaction, error)
	FetchAllCategories(ctx context.Context, userID int64, includeDeleted bool) ([]models.Category, error)
	FetchCategoryByID(ctx context.Context, userID int64, id int64, includeDeleted bool) (*models.Category, error)
	InsertTransaction(ctx context.Context, userID int64, req *models.TransactionReq) (int64, error)
	InsertTransfer(ctx context.Context, userID int64, req *models.TransferReq) (int64, error)
	InsertCategory(ctx context.Context, userID int64, req *models.CategoryReq) (int64, error)
	UpdateTransaction(ctx context.Context, userID int64, id int64, req *models.TransactionReq) (int64, error)
	UpdateCategory(ctx context.Context, userID int64, id int64, req *models.CategoryReq) (int64, error)
	DeleteTransaction(ctx context.Context, userID int64, id int64) error
	DeleteTransfer(ctx context.Context, userID int64, id int64) error
	DeleteCategory(ctx context.Context, userID int64, id int64) error
	RestoreTransaction(ctx context.Context, userID int64, id int64) error
	RestoreCategory(ctx context.Context, userID int64, id int64) error
	RestoreCategoryName(ctx context.Context, userID int64, id int64) error
	FetchTransactionTemplatesPaginated(ctx context.Context, userID int64, p utils.PaginationParams) ([]models.TransactionTemplate, *utils.Paginator, error)
	FetchTransactionTemplateByID(ctx context.Context, userID int64, id int64) (*models.TransactionTemplate, error)
	InsertTransactionTemplate(ctx context.Context, userID int64, req *models.TransactionTemplateReq) (int64, error)
	UpdateTransactionTemplate(ctx context.Context, userID, id int64, req *models.TransactionTemplateReq) (int64, error)
	ToggleTransactionTemplateActiveState(ctx context.Context, userID int64, id int64) error
	DeleteTransactionTemplate(ctx context.Context, userID int64, id int64) error
	GetTransactionTemplateCount(ctx context.Context, userID int64) (int64, error)
	GetTemplatesReadyToRun(ctx context.Context, tx *gorm.DB) ([]*models.TransactionTemplate, error)
	ProcessTemplate(ctx context.Context, template *models.TransactionTemplate) error
	FetchAllCategoryGroups(ctx context.Context, userID int64) ([]models.CategoryGroup, error)
	FetchAllCategoriesWithGroups(ctx context.Context, userID int64) ([]models.CategoryOrGroup, error)
	FetchCategoryGroupByID(ctx context.Context, userID int64, id int64) (*models.CategoryGroup, error)
	InsertCategoryGroup(ctx context.Context, userID int64, req *models.CategoryGroupReq) (int64, error)
	UpdateCategoryGroup(ctx context.Context, userID int64, id int64, req *models.CategoryGroupReq) (int64, error)
	DeleteCategoryGroup(ctx context.Context, userID int64, id int64) error
}

type TransactionService struct {
	repo              repositories.TransactionRepositoryInterface
	accRepo           repositories.AccountRepositoryInterface
	settingsRepo      repositories.SettingsRepositoryInterface
	loggingRepo       repositories.LoggingRepositoryInterface
	jobDispatcher     jobqueue.JobDispatcher
	currencyConverter finance.CurrencyManager
}

func NewTransactionService(
	repo *repositories.TransactionRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
	currencyConverter finance.CurrencyManager,
) *TransactionService {
	return &TransactionService{
		repo:              repo,
		accRepo:           accRepo,
		settingsRepo:      settingsRepo,
		loggingRepo:       loggingRepo,
		jobDispatcher:     jobDispatcher,
		currencyConverter: currencyConverter,
	}
}

var _ TransactionServiceInterface = (*TransactionService)(nil)

func (s *TransactionService) updateAccountBalance(ctx context.Context, tx *gorm.DB, account *models.Account, txnDate time.Time, direction string, amount decimal.Decimal) error {
	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, account.ID, txnDate, account.Currency); err != nil {
		return err
	}

	column := map[string]string{
		"expense": "cash_outflows",
		"income":  "cash_inflows",
	}[direction]

	if err := s.accRepo.AddToDailyBalance(ctx, tx, account.ID, txnDate, column, amount.Round(4)); err != nil {
		return err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(
		ctx,
		tx,
		account.UserID,
		account.ID,
		account.Currency,
		txnDate.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) FetchTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeDeleted bool, accountID *int64) ([]models.Transaction, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountTransactions(ctx, nil, userID, p.Filters, includeDeleted, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindTransactions(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeDeleted, accountID)
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

func (s *TransactionService) FetchTransfersPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeDeleted bool, accountID *int64) ([]models.Transfer, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountTransfers(ctx, nil, userID, includeDeleted, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindTransfers(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, includeDeleted, accountID)
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

func (s *TransactionService) FetchTransactionByID(ctx context.Context, userID int64, id int64, includeDeleted bool) (*models.Transaction, error) {

	record, err := s.repo.FindTransactionByID(ctx, nil, id, userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) FetchAllCategories(ctx context.Context, userID int64, includeDeleted bool) ([]models.Category, error) {

	categories, err := s.repo.FindAllCategories(ctx, nil, &userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *TransactionService) FetchCategoryByID(ctx context.Context, userID int64, id int64, includeDeleted bool) (*models.Category, error) {

	record, err := s.repo.FindCategoryByID(ctx, nil, id, &userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) validateInvestmentBalance(ctx context.Context, tx *gorm.DB, account *models.Account, userID int64, latestBalance *models.Balance, cashDelta decimal.Decimal) error {
	totalInvestmentValue, negativeValue, err := s.currencyConverter.ConvertInvestmentValueToAccountCurrency(ctx, tx, account.ID, userID, account.Currency)
	if err != nil {
		return fmt.Errorf("failed to calculate total investment value: %w", err)
	}

	// Adjust balance by adding back unrealized losses
	adjustedBalance := latestBalance.EndBalance.Add(negativeValue.Abs())

	// Calculate what the balance would be after this transaction
	resultingBalance := adjustedBalance.Add(cashDelta)

	// Calculate available cash after accounting for investments
	availableCashAfterTransaction := resultingBalance.Sub(totalInvestmentValue)

	if availableCashAfterTransaction.LessThan(decimal.Zero) && account.AccountType.Classification != "liability" {
		return fmt.Errorf("insufficient funds: resulting available cash (%s) would be negative (balance: %s, invested: %s)",
			availableCashAfterTransaction.StringFixed(2),
			resultingBalance.StringFixed(2),
			totalInvestmentValue.StringFixed(2))
	}

	return nil
}

func (s *TransactionService) InsertTransaction(ctx context.Context, userID int64, req *models.TransactionReq) (int64, error) {

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

	if req.TransactionType == "expense" {
		latestBalance, err := s.accRepo.FindLatestBalance(ctx, tx, account.ID, userID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		if err := s.validateInvestmentBalance(ctx, tx, account, userID, latestBalance, req.Amount.Neg()); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	// pick the user's timezone from settings; fall back to UTC
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// block transactions before opening date
	openAsOf, err := s.accRepo.GetAccountOpeningAsOf(ctx, tx, account.ID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("account has no opening balance; set an opening balance first")
		}
		return 0, err
	}

	txDay := utils.LocalMidnightUTC(req.TxnDate, loc)
	openDay := utils.LocalMidnightUTC(openAsOf, loc)
	todayDay := utils.LocalMidnightUTC(time.Now(), loc)

	if txDay.Before(openDay) {
		tx.Rollback()
		return 0, fmt.Errorf(
			"transaction date (%s) cannot be before account opening date (%s)",
			txDay.Format("2006-01-02"), openDay.Format("2006-01-02"),
		)
	}
	if txDay.After(todayDay) {
		tx.Rollback()
		return 0, fmt.Errorf(
			"transaction date (%s) cannot be in the future (>%s)",
			txDay.Format("2006-01-02"), todayDay.Format("2006-01-02"),
		)
	}

	var category models.Category
	if req.CategoryID != nil {
		category, err = s.repo.FindCategoryByID(ctx, tx, *req.CategoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("can't find category with given id %w", err)
		}
	} else {
		category, err = s.repo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("can't find default category %w", err)
		}
	}

	tr := models.Transaction{
		UserID:          userID,
		AccountID:       account.ID,
		CategoryID:      &category.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         txDay,
		Description:     req.Description,
	}

	txnID, err := s.repo.InsertTransaction(ctx, tx, &tr)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := s.updateAccountBalance(ctx, tx, account, tr.TxnDate, tr.TransactionType, tr.Amount); err != nil {
		tx.Rollback()
		return 0, err
	}

	// forward-fill the balance chain when the txn is back-dated
	from := tr.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if from.Before(today) {

		from := tr.TxnDate.UTC().Truncate(24 * time.Hour)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if err := s.accRepo.FrontfillBalances(ctx, tx, account.ID, account.Currency, from); err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, account.ID, account.Currency, from, today); err != nil {
			tx.Rollback()
			return 0, err
		}

	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	amountString := tr.Amount.StringFixed(2)
	dateStr := tr.TxnDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", strconv.FormatInt(txnID, 10), changes, "id")
	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", tr.TransactionType, changes, "type")
	utils.CompareChanges("", dateStr, changes, "date")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", tr.Currency, changes, "currency")
	utils.CompareChanges("", category.Name, changes, "category")
	utils.CompareChanges("", utils.SafeString(tr.Description), changes, "description")

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return txnID, nil
}

func (s *TransactionService) InsertTransfer(ctx context.Context, userID int64, req *models.TransferReq) (int64, error) {

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

	fromAcc, err := s.accRepo.FindAccountByID(ctx, tx, req.SourceID, userID, true)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find source account %w", err)
	}

	if fromAcc.AccountType.Classification == "asset" && fromAcc.Balance.EndBalance.LessThan(req.Amount) {
		tx.Rollback()
		return 0, fmt.Errorf("%w: account %s balance=%s, requested=%s",
			errors.New("insufficient funds"),
			fromAcc.Name,
			fromAcc.Balance.EndBalance.StringFixed(2),
			req.Amount.StringFixed(2),
		)
	}

	// Check against total cash invested
	totalInvestmentValue, negativeValue, err := s.currencyConverter.ConvertInvestmentValueToAccountCurrency(ctx, tx, fromAcc.ID, userID, fromAcc.Currency)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to calculate total investment value: %w", err)
	}

	// Adjust balance by adding back unrealized losses
	adjustedBalance := fromAcc.Balance.EndBalance.Add(negativeValue.Abs())

	// Calculate what the balance would be after withdrawing req.Amount
	resultingBalance := adjustedBalance.Sub(req.Amount)

	// Calculate available cash after accounting for investments
	availableCashAfterTransfer := resultingBalance.Sub(totalInvestmentValue)

	if availableCashAfterTransfer.LessThan(decimal.Zero) && fromAcc.AccountType.Classification != "liability" {
		tx.Rollback()
		return 0, fmt.Errorf("insufficient funds: resulting available cash (%s) would be negative in %s (balance: %s, invested: %s)",
			availableCashAfterTransfer.StringFixed(2),
			fromAcc.Name,
			resultingBalance.StringFixed(2),
			totalInvestmentValue.StringFixed(2))
	}

	toAcc, err := s.accRepo.FindAccountByID(ctx, tx, req.DestinationID, userID, false)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find destination account %w", err)
	}

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	t := req.CreatedAt
	if t.IsZero() {
		t = time.Now().UTC()
	}

	txDate := utils.LocalMidnightUTC(t, loc)

	outflow := models.Transaction{
		UserID:          userID,
		AccountID:       fromAcc.ID,
		TransactionType: "expense",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         txDate,
		Description:     req.Notes,
		IsTransfer:      true,
	}

	if _, err := s.repo.InsertTransaction(ctx, tx, &outflow); err != nil {
		tx.Rollback()
		return 0, err
	}

	inflow := models.Transaction{
		UserID:          userID,
		AccountID:       toAcc.ID,
		TransactionType: "income",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         txDate,
		Description:     req.Notes,
		IsTransfer:      true,
	}

	if _, err := s.repo.InsertTransaction(ctx, tx, &inflow); err != nil {
		tx.Rollback()
		return 0, err
	}

	transfer := models.Transfer{
		UserID:               userID,
		TransactionInflowID:  inflow.ID,
		TransactionOutflowID: outflow.ID,
		Amount:               req.Amount,
		Currency:             models.DefaultCurrency,
		Status:               "success",
		Notes:                req.Notes,
		CreatedAt:            req.CreatedAt,
	}

	trID, err := s.repo.InsertTransfer(ctx, tx, &transfer)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Update balances for both accounts
	if err := s.updateAccountBalance(ctx, tx, fromAcc, outflow.TxnDate, "expense", outflow.Amount); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := s.updateAccountBalance(ctx, tx, toAcc, inflow.TxnDate, "income", inflow.Amount); err != nil {
		tx.Rollback()
		return 0, err
	}

	from := txDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Frontfill and update snapshots for both accounts
	if err := s.accRepo.FrontfillBalances(ctx, tx, fromAcc.ID, fromAcc.Currency, from); err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, fromAcc.ID, fromAcc.Currency, from, today); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, toAcc.ID, toAcc.Currency, from); err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, toAcc.ID, toAcc.Currency, from, today); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(trID, 10), changes, "id")
	utils.CompareChanges("", fromAcc.Name, changes, "from")
	utils.CompareChanges("", toAcc.Name, changes, "to")
	utils.CompareChanges("", req.Amount.StringFixed(2), changes, "amount")
	utils.CompareChanges("", transfer.Currency, changes, "currency")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "transfer",
		Description: req.Notes,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	return trID, nil
}

func (s *TransactionService) InsertCategory(ctx context.Context, userID int64, req *models.CategoryReq) (int64, error) {

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

	cat, err := s.repo.FindCategoryByName(ctx, tx, req.Classification, &userID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	rec := models.Category{
		UserID:         &userID,
		Classification: req.Classification,
		DisplayName:    req.DisplayName,
		Name:           utils.NormalizeName(req.DisplayName),
		ParentID:       &cat.ID,
		IsDefault:      false,
	}

	catID, err := s.repo.InsertCategory(ctx, tx, &rec)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(catID, 10), changes, "id")
	utils.CompareChanges("", rec.DisplayName, changes, "name")
	utils.CompareChanges("", rec.Classification, changes, "classification")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "category",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	return catID, nil
}

func (s *TransactionService) UpdateTransaction(ctx context.Context, userID int64, id int64, req *models.TransactionReq) (int64, error) {
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

	// Load existing transaction
	exTr, err := s.repo.FindTransactionByID(ctx, tx, id, userID, false)
	if err != nil {
		return 0, fmt.Errorf("can't find transaction with given id %w", err)
	}
	if exTr.IsAdjustment {
		return 0, errors.New("can't edit a manual adjustment transaction")
	}

	// Load old account & category (for logs)
	oldAccount, err := s.accRepo.FindAccountByID(ctx, tx, exTr.AccountID, userID, false)
	if err != nil {
		return 0, fmt.Errorf("can't find existing account: %w", err)
	}
	var oldCategory models.Category
	if exTr.CategoryID != nil {
		oldCategory, err = s.repo.FindCategoryByID(ctx, tx, *exTr.CategoryID, &userID, true)
		if err != nil {
			return 0, fmt.Errorf("can't find existing category with given id %w", err)
		}
	}

	// Resolve new account & category
	newAccount, err := s.accRepo.FindAccountByID(ctx, tx, req.AccountID, userID, false)
	if err != nil {
		return 0, fmt.Errorf("can't find account with given id %w", err)
	}
	var newCategory models.Category
	if req.CategoryID != nil {
		newCategory, err = s.repo.FindCategoryByID(ctx, tx, *req.CategoryID, &userID, false)
		if err != nil {
			return 0, fmt.Errorf("can't find new category with given id %w", err)
		}
	} else {
		newCategory, err = s.repo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
		if err != nil {
			return 0, fmt.Errorf("can't find default category %w", err)
		}
	}

	var oldEffect, newEffect decimal.Decimal

	if exTr.TransactionType == "expense" {
		oldEffect = exTr.Amount.Neg()
	} else {
		oldEffect = exTr.Amount
	}

	if req.TransactionType == "expense" {
		newEffect = req.Amount.Neg()
	} else {
		newEffect = req.Amount
	}

	netChange := newEffect.Sub(oldEffect)

	// If net change is negative (balance going down), validate
	if netChange.IsNegative() {
		latestBalance, err := s.accRepo.FindLatestBalance(ctx, tx, newAccount.ID, userID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		if err := s.validateInvestmentBalance(ctx, tx, newAccount, userID, latestBalance, netChange); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// Block before opening
	openAsOf, err := s.accRepo.GetAccountOpeningAsOf(ctx, tx, newAccount.ID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("account has no opening balance; set an opening balance first")
		}
		return 0, err
	}

	newDay := utils.LocalMidnightUTC(req.TxnDate, loc)
	oldDay := utils.LocalMidnightUTC(exTr.TxnDate, loc)
	openDay := utils.LocalMidnightUTC(openAsOf, loc)
	todayDay := utils.LocalMidnightUTC(time.Now(), loc)

	if newDay.Before(openDay) {
		tx.Rollback()
		return 0, fmt.Errorf(
			"transaction date (%s) cannot be before account opening date (%s)",
			newDay.Format("2006-01-02"), openDay.Format("2006-01-02"),
		)
	}

	if newDay.After(todayDay) {
		tx.Rollback()
		return 0, fmt.Errorf(
			"transaction date (%s) cannot be in the future (>%s)",
			newDay.Format("2006-01-02"), todayDay.Format("2006-01-02"),
		)
	}

	// Update the transaction
	tr := models.Transaction{
		ID:              exTr.ID,
		UserID:          userID,
		AccountID:       newAccount.ID,
		CategoryID:      &newCategory.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Currency:        exTr.Currency,
		TxnDate:         newDay,
		Description:     req.Description,
	}
	txnID, err := s.repo.UpdateTransaction(ctx, tx, tr)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Adjust balances

	// Reverse old, apply new
	if !exTr.Amount.IsZero() {
		if err := s.updateAccountBalance(ctx, tx, oldAccount, oldDay, exTr.TransactionType, exTr.Amount.Neg()); err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	if !tr.Amount.IsZero() {
		if err := s.updateAccountBalance(ctx, tx, newAccount, newDay, tr.TransactionType, tr.Amount); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Determine the earliest affected date
	earliestDate := oldDay
	if newDay.Before(earliestDate) {
		earliestDate = newDay
	}

	// If account changed, we need to update both accounts
	if oldAccount.ID != newAccount.ID {
		// Update old account from old date forward
		if err := s.accRepo.FrontfillBalances(ctx, tx, oldAccount.ID, oldAccount.Currency, oldDay); err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, oldAccount.ID, oldAccount.Currency, oldDay, today); err != nil {
			tx.Rollback()
			return 0, err
		}

		// Update new account from new date forward
		if err := s.accRepo.FrontfillBalances(ctx, tx, newAccount.ID, newAccount.Currency, newDay); err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, newAccount.ID, newAccount.Currency, newDay, today); err != nil {
			tx.Rollback()
			return 0, err
		}
	} else {
		// Same account - update from earliest affected date forward
		if err := s.accRepo.FrontfillBalances(ctx, tx, newAccount.ID, newAccount.Currency, earliestDate); err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, newAccount.ID, newAccount.Currency, earliestDate, today); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(txnID, 10), changes, "id")
	utils.CompareChanges(oldAccount.Name, newAccount.Name, changes, "account")
	utils.CompareChanges(exTr.TransactionType, tr.TransactionType, changes, "type")
	utils.CompareDateChange(&exTr.TxnDate, &tr.TxnDate, changes, "date")
	utils.CompareDecimalChange(&exTr.Amount, &tr.Amount, changes, "amount", 2)
	utils.CompareChanges(exTr.Currency, tr.Currency, changes, "currency")
	utils.CompareChanges(oldCategory.Name, newCategory.Name, changes, "category")
	utils.CompareChanges(utils.SafeString(exTr.Description), utils.SafeString(tr.Description), changes, "description")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "update",
			Category:    "transaction",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return 0, err
		}
	}

	return txnID, nil
}

func (s *TransactionService) UpdateCategory(ctx context.Context, userID int64, id int64, req *models.CategoryReq) (int64, error) {

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

	exCat, err := s.repo.FindCategoryByID(ctx, tx, id, &userID, false)
	if err != nil {
		return 0, fmt.Errorf("can't find category with given id %w", err)
	}

	if exCat.IsDefault && (exCat.Classification != req.Classification) {
		return 0, errors.New("can't edit some parts of a default category")
	}

	cat := models.Category{
		ID:             exCat.ID,
		UserID:         &userID,
		Classification: req.Classification,
		DisplayName:    req.DisplayName,
	}

	catID, err := s.repo.UpdateCategory(ctx, tx, cat)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()

	utils.CompareChanges("", strconv.FormatInt(catID, 10), changes, "id")
	utils.CompareChanges(exCat.DisplayName, cat.DisplayName, changes, "name")
	utils.CompareChanges(exCat.Classification, cat.Classification, changes, "classification")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "update",
			Category:    "category",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return catID, nil
}

func (s *TransactionService) DeleteTransaction(ctx context.Context, userID int64, id int64) error {

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

	// Load the transaction + relations
	tr, err := s.repo.FindTransactionByID(ctx, tx, id, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find transaction with given id %w", err)
	}

	account, err := s.accRepo.FindAccountByID(ctx, tx, tr.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	// If deleting an income, balance will go down
	if tr.TransactionType == "income" {
		latestBalance, err := s.accRepo.FindLatestBalance(ctx, tx, account.ID, userID)
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := s.validateInvestmentBalance(ctx, tx, account, userID, latestBalance, tr.Amount.Neg()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Delete transaction
	if err := s.repo.DeleteTransaction(ctx, tx, tr.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	var category models.Category
	if tr.CategoryID != nil {
		cat, err := s.repo.FindCategoryByID(ctx, tx, *tr.CategoryID, &userID, true)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find category with given id %w", err)
		}
		category = cat
	}

	// Reverse the original cash effect on the account
	if err := s.updateAccountBalance(ctx, tx, account, tr.TxnDate, tr.TransactionType, tr.Amount.Neg()); err != nil {
		tx.Rollback()
		return err
	}

	from := tr.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err := s.accRepo.FrontfillBalances(ctx, tx, account.ID, account.Currency, from); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, account.ID, account.Currency, from, today); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()

	utils.CompareChanges("", strconv.FormatInt(tr.ID, 10), changes, "id")
	utils.CompareChanges(account.Name, "", changes, "account")
	utils.CompareChanges(tr.TransactionType, "", changes, "type")
	utils.CompareDateChange(&tr.TxnDate, nil, changes, "date")
	utils.CompareDecimalChange(&tr.Amount, nil, changes, "amount", 2)
	utils.CompareChanges(tr.Currency, "", changes, "currency")
	utils.CompareChanges(utils.SafeString(&category.Name), "", changes, "category")
	utils.CompareChanges(utils.SafeString(tr.Description), "", changes, "description")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "delete",
			Category:    "transaction",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteTransfer(ctx context.Context, userID int64, id int64) error {

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

	// Load the transfer
	transfer, err := s.repo.FindTransferByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find transfer with given id %w", err)
	}

	// Load associated transactions
	inflow, err := s.repo.FindTransactionByID(ctx, tx, transfer.TransactionInflowID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}

	outflow, err := s.repo.FindTransactionByID(ctx, tx, transfer.TransactionOutflowID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find outflow transaction with given id %w", err)
	}

	// Load accounts
	fromAcc, err := s.accRepo.FindAccountByID(ctx, tx, outflow.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}
	toAcc, err := s.accRepo.FindAccountByID(ctx, tx, inflow.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find destination account %w", err)
	}

	if err := utils.ValidateAccount(fromAcc, "source"); err != nil {
		return err
	}
	if err := utils.ValidateAccount(toAcc, "destination"); err != nil {
		return err
	}

	// Check if removing from destination would violate investment constraint
	latestToBalance, err := s.accRepo.FindLatestBalance(ctx, tx, toAcc.ID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.validateInvestmentBalance(ctx, tx, toAcc, userID, latestToBalance, inflow.Amount.Neg()); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.updateAccountBalance(ctx, tx, fromAcc, outflow.TxnDate, "expense", outflow.Amount.Neg()); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.updateAccountBalance(ctx, tx, toAcc, outflow.TxnDate, "income", outflow.Amount.Neg()); err != nil {
		tx.Rollback()
		return err
	}

	from := outflow.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// frontfill from the transfer date forward
	if err := s.accRepo.FrontfillBalances(ctx, tx, fromAcc.ID, fromAcc.Currency, from); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, fromAcc.ID, fromAcc.Currency, from, today); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.accRepo.FrontfillBalances(ctx, tx, toAcc.ID, toAcc.Currency, from); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, toAcc.ID, toAcc.Currency, from, today); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transfer
	if err := s.repo.DeleteTransfer(ctx, tx, transfer.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transactions
	if err := s.repo.DeleteTransaction(ctx, tx, inflow.ID, userID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.repo.DeleteTransaction(ctx, tx, outflow.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log synthetic transfer deletion
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(transfer.ID, 10), changes, "id")
	utils.CompareChanges(fromAcc.Name, "", changes, "from")
	utils.CompareChanges(toAcc.Name, "", changes, "to")
	utils.CompareChanges(transfer.Amount.StringFixed(2), "", changes, "amount")
	utils.CompareChanges(transfer.Currency, "", changes, "currency")
	utils.CompareChanges(utils.SafeString(transfer.Notes), "", changes, "description")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "delete",
			Category:    "transfer",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteCategory(ctx context.Context, userID int64, id int64) error {

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

	cat, err := s.repo.FindCategoryByID(ctx, tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category with given id: %w", err)
	}

	// Check if category is part of any group
	inGroup, err := s.repo.IsCategoryInGroup(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check category group membership: %w", err)
	}
	if inGroup {
		tx.Rollback()
		return fmt.Errorf("cannot delete category: it is part of one or more category groups")
	}

	alreadySoftDeleted := cat.DeletedAt != nil
	var deleteType string

	switch {
	case !alreadySoftDeleted:
		// Archive first
		if err := s.repo.ArchiveCategory(ctx, tx, cat.ID, userID); err != nil {
			tx.Rollback()
			return err
		}
		deleteType = "soft"

	case !cat.IsDefault && alreadySoftDeleted:
		// Non-default category, already archived -> try permanent delete
		cnt, err := s.repo.CountActiveTransactionsForCategory(ctx, tx, userID, cat.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
		if cnt > 0 {
			tx.Rollback()
			return fmt.Errorf("cannot permanently delete category: %d active transactions still reference it", cnt)
		}
		if err := s.repo.DeleteCategory(ctx, tx, cat.ID, userID); err != nil {
			tx.Rollback()
			return err
		}
		deleteType = "hard"
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(cat.ID, 10), changes, "id")
	utils.CompareChanges(deleteType, "", changes, "delete_type")
	utils.CompareChanges(cat.DisplayName, "", changes, "name")
	utils.CompareChanges(cat.Classification, "", changes, "classification")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "delete",
			Category:    "category",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) RestoreTransaction(ctx context.Context, userID int64, id int64) error {

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

	// Load the transaction
	tr, err := s.repo.FindTransactionByID(ctx, tx, id, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}
	if tr.DeletedAt == nil {
		tx.Rollback()
		return fmt.Errorf("transaction is not deleted")
	}

	// Load account
	acc, err := s.accRepo.FindAccountByID(ctx, tx, tr.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account for transaction %w", err)
	}

	// If restoring an expense, balance will go down
	if tr.TransactionType == "expense" {
		latestBalance, err := s.accRepo.FindLatestBalance(ctx, tx, acc.ID, userID)
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := s.validateInvestmentBalance(ctx, tx, acc, userID, latestBalance, tr.Amount.Neg()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Re-apply og cash effect
	signed := func(tt string, amt decimal.Decimal) decimal.Decimal {
		switch strings.ToLower(tt) {
		case "expense":
			return amt.Neg()
		default:
			return amt
		}
	}
	origEffect := signed(tr.TransactionType, tr.Amount)

	// Reverse balances
	if !origEffect.IsZero() {
		dir := map[bool]string{true: "expense", false: "income"}[origEffect.IsNegative()]

		if err := s.updateAccountBalance(ctx, tx, acc, tr.TxnDate, dir, origEffect.Abs()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Unmark as soft deleted
	if err := s.repo.RestoreTransaction(ctx, tx, tr.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(tr.ID, 10), changes, "id")
	utils.CompareChanges("", acc.Name, changes, "account")
	utils.CompareChanges("", tr.Amount.StringFixed(2), changes, "amount")
	utils.CompareChanges("", tr.Currency, changes, "currency")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "restore",
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) RestoreCategory(ctx context.Context, userID int64, id int64) error {

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

	// Load the record
	cat, err := s.repo.FindCategoryByID(ctx, tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing category with given id %w", err)
	}
	if cat.DeletedAt == nil {
		tx.Rollback()
		return fmt.Errorf("category is not deleted")
	}

	// Unmark as soft deleted
	if err := s.repo.RestoreCategory(ctx, tx, cat.ID, &userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(cat.ID, 10), changes, "id")
	utils.CompareChanges("", cat.DisplayName, changes, "name")
	utils.CompareChanges("", cat.Classification, changes, "classification")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "restore",
		Category:    "category",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) RestoreCategoryName(ctx context.Context, userID int64, id int64) error {

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

	// Load the record
	cat, err := s.repo.FindCategoryByID(ctx, tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing category with given id %w", err)
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(cat.ID, 10), changes, "id")
	utils.CompareChanges(utils.NormalizeName(cat.DisplayName), cat.Name, changes, "name")

	if err := s.repo.RestoreCategoryName(ctx, tx, cat.ID, &userID, cat.Name); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "restore",
		Category:    "category",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) FetchTransactionTemplatesPaginated(ctx context.Context, userID int64, p utils.PaginationParams) ([]models.TransactionTemplate, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountTransactionTemplates(ctx, nil, userID, false)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindTransactionTemplates(ctx, nil, userID, offset, p.RowsPerPage)
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

func (s *TransactionService) FetchTransactionTemplateByID(ctx context.Context, userID int64, id int64) (*models.TransactionTemplate, error) {

	record, err := s.repo.FindTransactionTemplateByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) InsertTransactionTemplate(ctx context.Context, userID int64, req *models.TransactionTemplateReq) (int64, error) {
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

	category, err := s.repo.FindCategoryByID(ctx, tx, req.CategoryID, &userID, false)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find category with given id %w", err)
	}

	// pick the user's timezone from settings; fall back to UTC
	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	firstRun := utils.LocalMidnightUTC(req.NextRunAt, loc)

	firstValidDay := time.Now().UTC().Truncate(24 * time.Hour)

	if firstRun.Before(firstValidDay) {
		tx.Rollback()
		return 0, fmt.Errorf(
			"first iteration of template cannot be executed in the same day (%s)",
			firstValidDay.Format("2006-01-02"),
		)
	}

	if req.MaxRuns != nil {
		if *req.MaxRuns < 0 || *req.MaxRuns > 99999 {
			tx.Rollback()
			return 0, fmt.Errorf("max runs out of bounds %w", err)
		}
	}

	var endDate *time.Time
	if req.EndDate != nil {
		e := utils.LocalMidnightUTC(*req.EndDate, loc)
		endDate = &e
	}

	tp := models.TransactionTemplate{
		Name:            req.Name,
		UserID:          userID,
		AccountID:       account.ID,
		CategoryID:      category.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Frequency:       strings.ToLower(req.Frequency),
		NextRunAt:       firstRun,
		EndDate:         endDate,
		MaxRuns:         req.MaxRuns,
		RunCount:        0,
		IsActive:        true,
	}

	tpID, err := s.repo.InsertTransactionTemplate(ctx, tx, &tp)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	amountString := tp.Amount.StringFixed(2)
	firstRunStr := tp.NextRunAt.UTC().Format(time.RFC3339)

	utils.CompareChanges("", strconv.FormatInt(tpID, 10), changes, "id")
	utils.CompareChanges("", tp.Name, changes, "name")
	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", category.Name, changes, "category")
	utils.CompareChanges("", tp.TransactionType, changes, "type")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", firstRunStr, changes, "first_run")

	if tp.EndDate != nil {
		endDateStr := tp.EndDate.UTC().Format(time.RFC3339)
		utils.CompareChanges("", endDateStr, changes, "end_date")
	}

	if tp.MaxRuns != nil {
		maxRunsStr := strconv.FormatInt(int64(*tp.MaxRuns), 10)
		utils.CompareChanges("", maxRunsStr, changes, "max_runs")
	}

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "txn_template",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return tpID, nil
}

func (s *TransactionService) UpdateTransactionTemplate(ctx context.Context, userID, id int64, req *models.TransactionTemplateReq) (int64, error) {

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

	changes := utils.InitChanges()

	// Load existing transaction template
	exTp, err := s.repo.FindTransactionTemplateByID(ctx, tx, id, userID)
	if err != nil {
		return 0, fmt.Errorf("can't find transaction template with given id %w", err)
	}

	// Prevent updates if template has completed its runs
	if exTp.MaxRuns != nil && exTp.RunCount >= *exTp.MaxRuns {
		tx.Rollback()
		return 0, fmt.Errorf("cannot update completed template (max runs reached)")
	}

	if exTp.EndDate != nil && time.Now().UTC().After(*exTp.EndDate) {
		tx.Rollback()
		return 0, fmt.Errorf("cannot update expired template (end date passed)")
	}

	// pick the user's timezone from settings; fall back to UTC
	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	nextRun := utils.LocalMidnightUTC(req.NextRunAt, loc)

	firstValidDay := time.Now().UTC().Truncate(24 * time.Hour)
	if nextRun.Before(firstValidDay) {
		tx.Rollback()
		return 0, fmt.Errorf("next run cannot be today or earlier (%s)", firstValidDay.Format("2006-01-02"))
	}

	if req.MaxRuns != nil {
		if *req.MaxRuns < 0 || *req.MaxRuns > 99999 {
			tx.Rollback()
			return 0, fmt.Errorf("max runs out of bounds %w", err)
		}
	}

	var endDate *time.Time
	if req.EndDate != nil {
		e := utils.LocalMidnightUTC(*req.EndDate, loc)
		endDate = &e
	}

	tp := models.TransactionTemplate{
		ID:              exTp.ID,
		Name:            req.Name,
		UserID:          userID,
		AccountID:       exTp.AccountID,
		CategoryID:      exTp.CategoryID,
		TransactionType: strings.ToLower(exTp.TransactionType),
		Amount:          req.Amount,
		Frequency:       exTp.Frequency,
		NextRunAt:       nextRun,
		EndDate:         endDate,
		MaxRuns:         req.MaxRuns,
		RunCount:        exTp.RunCount,
		IsActive:        exTp.IsActive,
	}

	tpID, err := s.repo.UpdateTransactionTemplate(ctx, tx, tp, false)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	exAmountString := exTp.Amount.StringFixed(2)
	amountString := tp.Amount.StringFixed(2)
	exNextRunStr := exTp.NextRunAt.UTC().Format(time.RFC3339)
	nextRunStr := tp.NextRunAt.UTC().Format(time.RFC3339)
	exIsActiveStr := strconv.FormatBool(exTp.IsActive)
	isActiveStr := strconv.FormatBool(tp.IsActive)

	utils.CompareChanges(exTp.Name, tp.Name, changes, "name")
	utils.CompareChanges(exAmountString, amountString, changes, "amount")
	utils.CompareChanges(exNextRunStr, nextRunStr, changes, "next_run")
	utils.CompareChanges(exIsActiveStr, isActiveStr, changes, "is_active")

	if tp.EndDate != nil {
		var exEndDateStr string
		if exTp.EndDate != nil {
			exEndDateStr = tp.EndDate.UTC().Format(time.RFC3339)
		} else {
			exEndDateStr = ""
		}
		endDateStr := tp.EndDate.UTC().Format(time.RFC3339)
		utils.CompareChanges(exEndDateStr, endDateStr, changes, "end_date")
	}

	if tp.MaxRuns != nil {
		var exMaxRunsStr string
		if exTp.MaxRuns != nil {
			exMaxRunsStr = tp.EndDate.UTC().Format(time.RFC3339)
		} else {
			exMaxRunsStr = ""
		}
		maxRunsStr := strconv.FormatInt(int64(*tp.MaxRuns), 10)
		utils.CompareChanges(exMaxRunsStr, maxRunsStr, changes, "max_runs")
	}

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "update",
		Category:    "txn_template",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return tpID, nil
}

func (s *TransactionService) ToggleTransactionTemplateActiveState(ctx context.Context, userID int64, id int64) error {

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

	// Load record to confirm it exists
	exTp, err := s.repo.FindTransactionTemplateByID(ctx, tx, id, userID)
	if err != nil {
		return fmt.Errorf("can't find transaction template with given id %w", err)
	}

	// Prevent enabling if template has completed its runs
	if !exTp.IsActive {
		if exTp.MaxRuns != nil && exTp.RunCount >= *exTp.MaxRuns {
			tx.Rollback()
			return fmt.Errorf("cannot enable completed template (max runs reached)")
		}

		if exTp.EndDate != nil && time.Now().UTC().After(*exTp.EndDate) {
			tx.Rollback()
			return fmt.Errorf("cannot enable expired template (end date passed)")
		}
	}

	tp := models.TransactionTemplate{
		ID:       exTp.ID,
		UserID:   userID,
		IsActive: !exTp.IsActive,
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(tp.ID, 10), changes, "id")
	utils.CompareChanges(strconv.FormatBool(exTp.IsActive), strconv.FormatBool(tp.IsActive), changes, "is_active")

	_, err = s.repo.UpdateTransactionTemplate(ctx, tx, tp, true)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,

			Event:       "update",
			Category:    "txn_template",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteTransactionTemplate(ctx context.Context, userID int64, id int64) error {
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

	// Confirm existence
	tp, err := s.repo.FindTransactionTemplateByID(ctx, tx, id, userID)
	if err != nil {
		return fmt.Errorf("can't find transaction template with given id %w", err)
	}

	err = s.repo.DeleteTransactionTemplate(ctx, tx, tp.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	amountString := tp.Amount.StringFixed(2)
	firstRunStr := tp.NextRunAt.UTC().Format(time.RFC3339)

	utils.CompareChanges(tp.Name, "", changes, "name")
	utils.CompareChanges(tp.Account.Name, "", changes, "account")
	utils.CompareChanges(tp.Category.Name, "", changes, "category")
	utils.CompareChanges(tp.TransactionType, "", changes, "type")
	utils.CompareChanges(amountString, "", changes, "amount")
	utils.CompareChanges(firstRunStr, "", changes, "first_run")

	if tp.EndDate != nil {
		endDateStr := tp.EndDate.UTC().Format(time.RFC3339)
		utils.CompareChanges(endDateStr, "", changes, "end_date")
	}

	if tp.MaxRuns != nil {
		maxRunsStr := strconv.FormatInt(int64(*tp.MaxRuns), 10)
		utils.CompareChanges(maxRunsStr, "", changes, "max_runs")
	}

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "txn_template",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) GetTransactionTemplateCount(ctx context.Context, userID int64) (int64, error) {
	return s.repo.CountTransactionTemplates(ctx, nil, userID, true)
}

func (s *TransactionService) GetTemplatesReadyToRun(ctx context.Context, tx *gorm.DB) ([]*models.TransactionTemplate, error) {
	return s.repo.GetTemplatesReadyToRun(ctx, tx)
}

func (s *TransactionService) ProcessTemplate(ctx context.Context, template *models.TransactionTemplate) error {

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

	// Reload template to ensure it's still active and valid
	currentTemplate, err := s.repo.FindTransactionTemplateByID(ctx, tx, template.ID, template.UserID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("template not found: %w", err)
	}

	if !currentTemplate.IsActive {
		tx.Rollback()
		return fmt.Errorf("template is not active")
	}

	// Verify account still exists and is active
	acc, err := s.accRepo.FindAccountByID(ctx, tx, currentTemplate.AccountID, currentTemplate.UserID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("account not found: %w", err)
	}

	// Verify category still exists
	var categoryID int64
	_, err = s.repo.FindCategoryByID(ctx, tx, currentTemplate.CategoryID, &currentTemplate.UserID, false)
	if err != nil {
		cat, err := s.repo.FindCategoryByClassification(ctx, tx, "uncategorized", &currentTemplate.UserID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find default category %w", err)
		}
		categoryID = cat.ID
	} else {
		categoryID = currentTemplate.CategoryID
	}

	// Create the transaction
	desc := fmt.Sprintf("Auto: %s", currentTemplate.Name)
	txnReq := &models.TransactionReq{
		AccountID:       acc.ID,
		CategoryID:      &categoryID,
		TransactionType: currentTemplate.TransactionType,
		Amount:          currentTemplate.Amount,
		TxnDate:         currentTemplate.NextRunAt,
		Description:     &desc,
	}

	_, err = s.InsertTransaction(ctx, currentTemplate.UserID, txnReq)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Calculate next run date
	nextRun := utils.CalculateNextRun(currentTemplate.NextRunAt, currentTemplate.Frequency)
	now := time.Now().UTC()

	// Update template
	updates := map[string]interface{}{
		"last_run_at": now,
		"run_count":   currentTemplate.RunCount + 1,
		"next_run_at": nextRun,
	}

	// Check if we should deactivate
	shouldDeactivate := true
	switch {
	case currentTemplate.MaxRuns != nil && currentTemplate.RunCount+1 >= *currentTemplate.MaxRuns:
		// Max runs reached
	case currentTemplate.EndDate != nil && nextRun.After(*currentTemplate.EndDate):
		// End date passed
	default:
		shouldDeactivate = false
	}

	if shouldDeactivate {
		updates["is_active"] = false
	}

	if err := tx.Model(&models.TransactionTemplate{}).Where("id = ?", currentTemplate.ID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) FetchAllCategoryGroups(ctx context.Context, userID int64) ([]models.CategoryGroup, error) {

	categories, err := s.repo.FindAllCategoryGroups(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *TransactionService) FetchAllCategoriesWithGroups(ctx context.Context, userID int64) ([]models.CategoryOrGroup, error) {
	categories, groups, err := s.repo.FindAllCategoriesAndGroups(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	var results []models.CategoryOrGroup

	// Add individual categories
	for _, cat := range categories {
		results = append(results, models.CategoryOrGroup{
			ID:             cat.ID,
			Name:           cat.DisplayName,
			IsGroup:        false,
			Classification: cat.Classification,
			CategoryIDs:    []int64{cat.ID},
		})
	}

	// Add category groups
	for _, group := range groups {
		categoryIDs := make([]int64, len(group.Categories))
		for i, cat := range group.Categories {
			categoryIDs[i] = cat.ID
		}

		results = append(results, models.CategoryOrGroup{
			ID:             group.ID,
			Name:           group.Name,
			IsGroup:        true,
			Classification: group.Classification,
			CategoryIDs:    categoryIDs,
		})
	}

	return results, nil
}

func (s *TransactionService) FetchCategoryGroupByID(ctx context.Context, userID int64, id int64) (*models.CategoryGroup, error) {

	record, err := s.repo.FindCategoryGroupByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) InsertCategoryGroup(ctx context.Context, userID int64, req *models.CategoryGroupReq) (int64, error) {

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

	rec := models.CategoryGroup{
		UserID:         &userID,
		Classification: req.Classification,
		Name:           req.Name,
		Description:    req.Description,
	}

	groupingID, err := s.repo.InsertCategoryGroup(ctx, tx, &rec)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	categoryIDs, ok := req.SelectedCategories.([]interface{})
	if !ok || len(categoryIDs) == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("invalid or empty selected_categories")
	}

	for _, idVal := range categoryIDs {
		categoryID, _ := strconv.ParseInt(fmt.Sprint(idVal), 10, 64)

		// Validate category exists
		_, err := s.repo.FindCategoryByID(ctx, tx, categoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to validate category %d: %w", categoryID, err)
		}

		// Create the m:m relation
		if err := s.repo.InsertCategoryGroupMember(ctx, tx, groupingID, categoryID); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to link category %d: %w", categoryID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(groupingID, 10), changes, "id")
	utils.CompareChanges("", rec.Name, changes, "name")
	utils.CompareChanges("", rec.Classification, changes, "classification")
	utils.CompareChanges("", fmt.Sprintf("%d categories", len(categoryIDs)), changes, "categories_count")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "category_group",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	return groupingID, nil
}

func (s *TransactionService) UpdateCategoryGroup(ctx context.Context, userID int64, id int64, req *models.CategoryGroupReq) (int64, error) {

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

	// Get existing record
	exGroup, err := s.repo.FindCategoryGroupByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find category group with given id: %w", err)
	}

	rec := models.CategoryGroup{
		ID:             id,
		UserID:         &userID,
		Classification: req.Classification,
		Name:           req.Name,
		Description:    req.Description,
	}

	groupID, err := s.repo.UpdateCategoryGroup(ctx, tx, rec)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Delete existing category relations
	if err := s.repo.DeleteCategoryGroupMembers(ctx, tx, id); err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to clear existing categories: %w", err)
	}

	// Add new category relations
	categoryIDs, ok := req.SelectedCategories.([]interface{})
	if !ok || len(categoryIDs) == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("invalid or empty selected_categories")
	}

	for _, idVal := range categoryIDs {
		categoryID, _ := strconv.ParseInt(fmt.Sprint(idVal), 10, 64)

		// Validate category exists
		_, err := s.repo.FindCategoryByID(ctx, tx, categoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to validate category %d: %w", categoryID, err)
		}

		// Create the m:m relation
		if err := s.repo.InsertCategoryGroupMember(ctx, tx, id, categoryID); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to link category %d: %w", categoryID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(groupID, 10), changes, "id")
	utils.CompareChanges(exGroup.Name, rec.Name, changes, "name")
	utils.CompareChanges(exGroup.Classification, rec.Classification, changes, "classification")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "category_group",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return groupID, nil
}

func (s *TransactionService) DeleteCategoryGroup(ctx context.Context, userID int64, id int64) error {

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

	group, err := s.repo.FindCategoryGroupByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category group with given id: %w", err)
	}

	// Delete all category relations first
	if err := s.repo.DeleteCategoryGroupMembers(ctx, tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete category relations: %w", err)
	}

	// Delete the group itself
	if err := s.repo.DeleteCategoryGroup(ctx, tx, id, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(group.Name, "", changes, "name")
	utils.CompareChanges(group.Classification, "", changes, "classification")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "delete",
			Category:    "category_group",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}
