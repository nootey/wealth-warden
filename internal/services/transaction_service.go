package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

type TransactionService struct {
	Config         *config.Config
	Ctx            *DefaultServiceContext
	Repo           *repositories.TransactionRepository
	AccountService *AccountService
}

func NewTransactionService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.TransactionRepository,
	accService *AccountService,
) *TransactionService {
	return &TransactionService{
		Ctx:            ctx,
		Config:         cfg,
		Repo:           repo,
		AccountService: accService,
	}
}

func (s *TransactionService) FetchTransactionsPaginated(userID int64, p utils.PaginationParams, includeDeleted bool, accountID *int64) ([]models.Transaction, *utils.Paginator, error) {

	totalRecords, err := s.Repo.CountTransactions(userID, p.Filters, includeDeleted, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindTransactions(userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeDeleted, accountID)
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

func (s *TransactionService) FetchTransfersPaginated(userID int64, p utils.PaginationParams, includeDeleted bool) ([]models.Transfer, *utils.Paginator, error) {

	totalRecords, err := s.Repo.CountTransfers(userID, includeDeleted)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindTransfers(userID, offset, p.RowsPerPage, includeDeleted)
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

func (s *TransactionService) FetchTransactionByID(userID int64, id int64, includeDeleted bool) (*models.Transaction, error) {

	record, err := s.Repo.FindTransactionByID(nil, id, userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) FetchAllCategories(userID int64, includeDeleted bool) ([]models.Category, error) {

	categories, err := s.Repo.FindAllCategories(nil, &userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *TransactionService) FetchCategoryByID(userID int64, id int64, includeDeleted bool) (*models.Category, error) {

	record, err := s.Repo.FindCategoryByID(nil, id, &userID, includeDeleted)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) InsertTransaction(userID int64, req *models.TransactionReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	account, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	settings, err := s.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	// pick the user's timezone from settings; fall back to UTC
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// block transactions before opening date
	openAsOf, err := s.AccountService.Repo.GetAccountOpeningAsOf(tx, account.ID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("account has no opening balance; set an opening balance first")
		}
		return err
	}

	txDay := utils.LocalMidnightUTC(req.TxnDate, loc)
	openDay := utils.LocalMidnightUTC(openAsOf, loc)
	todayDay := utils.LocalMidnightUTC(time.Now(), loc)

	if txDay.Before(openDay) {
		tx.Rollback()
		return fmt.Errorf(
			"transaction date (%s) cannot be before account opening date (%s)",
			txDay.Format("2006-01-02"), openDay.Format("2006-01-02"),
		)
	}
	if txDay.After(todayDay) {
		tx.Rollback()
		return fmt.Errorf(
			"transaction date (%s) cannot be in the future (>%s)",
			txDay.Format("2006-01-02"), todayDay.Format("2006-01-02"),
		)
	}

	var category models.Category
	if req.CategoryID != nil {
		category, err = s.Repo.FindCategoryByID(tx, *req.CategoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find category with given id %w", err)
		}
	} else {
		category, err = s.Repo.FindCategoryByClassification(tx, "uncategorized", &userID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find default category %w", err)
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

	_, err = s.Repo.InsertTransaction(tx, &tr)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.AccountService.UpdateAccountCashBalance(tx, account, tr.TxnDate, tr.TransactionType, tr.Amount)
	if err != nil {
		tx.Rollback()
		return err
	}

	// forward-fill the balance chain when the txn is back-dated
	from := tr.TxnDate.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if from.Before(today) {
		if err := s.AccountService.FrontfillBalancesForAccount(
			tx, userID, account.ID, models.DefaultCurrency, from,
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	amountString := tr.Amount.StringFixed(2)
	dateStr := tr.TxnDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", tr.TransactionType, changes, "type")
	utils.CompareChanges("", dateStr, changes, "date")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", tr.Currency, changes, "currency")
	utils.CompareChanges("", category.Name, changes, "category")
	utils.CompareChanges("", utils.SafeString(tr.Description), changes, "description")

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	// Dispatch balance activity log
	var change decimal.Decimal
	if tr.TransactionType == "expense" {
		change = tr.Amount.Neg()
	} else {
		change = tr.Amount
	}

	if err := s.AccountService.LogBalanceChange(account, userID, change); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) InsertTransfer(userID int64, req *models.TransferReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	fromAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.SourceID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	if fromAccount.Balance.EndBalance.LessThan(req.Amount) {
		tx.Rollback()
		return fmt.Errorf("%w: account %s balance=%s, requested=%s",
			errors.New("insufficient funds"),
			fromAccount.Name,
			fromAccount.Balance.EndBalance.StringFixed(2),
			req.Amount.StringFixed(2),
		)
	}

	toAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.DestinationID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find destination account %w", err)
	}

	settings, err := s.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	t := req.CreatedAt
	if t.IsZero() {
		t = time.Now()
	}

	txDate := utils.LocalMidnightUTC(t, loc)

	outflow := models.Transaction{
		UserID:          userID,
		AccountID:       fromAccount.ID,
		TransactionType: "expense",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         txDate,
		Description:     req.Notes,
		IsTransfer:      true,
	}

	if _, err := s.Repo.InsertTransaction(tx, &outflow); err != nil {
		tx.Rollback()
		return err
	}

	inflow := models.Transaction{
		UserID:          userID,
		AccountID:       toAccount.ID,
		TransactionType: "income",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         txDate,
		Description:     req.Notes,
		IsTransfer:      true,
	}

	if _, err := s.Repo.InsertTransaction(tx, &inflow); err != nil {
		tx.Rollback()
		return err
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

	if _, err := s.Repo.InsertTransfer(tx, &transfer); err != nil {
		tx.Rollback()
		return err
	}

	// touch deltas for both accounts
	err = s.AccountService.UpdateBalancesForTransfer(tx, fromAccount, toAccount, txDate, req.Amount)
	if err != nil {
		tx.Rollback()
		return err
	}

	// frontfill from the transfer date forward
	if err := s.AccountService.FrontfillBalancesForAccount(tx, userID, fromAccount.ID, models.DefaultCurrency, txDate); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.AccountService.FrontfillBalancesForAccount(tx, userID, toAccount.ID, models.DefaultCurrency, txDate); err != nil {
		tx.Rollback()
		return err
	}

	// refresh snapshots for both accounts to today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err := s.AccountService.Repo.UpsertSnapshotsFromBalances(tx, userID, fromAccount.ID, models.DefaultCurrency, txDate, today); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.AccountService.Repo.UpsertSnapshotsFromBalances(tx, userID, toAccount.ID, models.DefaultCurrency, txDate, today); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", fromAccount.Name, changes, "from")
	utils.CompareChanges("", toAccount.Name, changes, "to")
	utils.CompareChanges("", req.Amount.StringFixed(2), changes, "amount")
	utils.CompareChanges("", transfer.Currency, changes, "currency")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "transfer",
		Description: req.Notes,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	// Log balance updates for both accounts
	if err := s.AccountService.LogBalanceChange(fromAccount, userID, req.Amount.Neg()); err != nil {
		return err
	}
	if err := s.AccountService.LogBalanceChange(toAccount, userID, req.Amount); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) InsertCategory(userID int64, req *models.CategoryReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	cat, err := s.Repo.FindCategoryByName(tx, req.Classification, &userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	rec := models.Category{
		UserID:         &userID,
		Classification: req.Classification,
		DisplayName:    req.DisplayName,
		Name:           utils.NormalizeName(req.DisplayName),
		ParentID:       &cat.ID,
		IsDefault:      false,
	}

	if _, err := s.Repo.InsertCategory(tx, &rec); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", rec.DisplayName, changes, "name")
	utils.CompareChanges("", rec.Classification, changes, "classification")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "category",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) UpdateTransaction(userID int64, id int64, req *models.TransactionReq) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load existing transaction
	exTr, err := s.Repo.FindTransactionByID(tx, id, userID, false)
	if err != nil {
		return fmt.Errorf("can't find transaction with given id %w", err)
	}
	if exTr.IsAdjustment {
		return errors.New("can't edit a manual adjustment transaction")
	}

	// Load old account & category (for logs)
	oldAccount, err := s.AccountService.Repo.FindAccountByID(tx, exTr.AccountID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find existing account: %w", err)
	}
	var oldCategory models.Category
	if exTr.CategoryID != nil {
		oldCategory, err = s.Repo.FindCategoryByID(tx, *exTr.CategoryID, &userID, true)
		if err != nil {
			return fmt.Errorf("can't find existing category with given id %w", err)
		}
	}

	// Resolve new account & category
	newAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}
	var newCategory models.Category
	if req.CategoryID != nil {
		newCategory, err = s.Repo.FindCategoryByID(tx, *req.CategoryID, &userID, false)
		if err != nil {
			return fmt.Errorf("can't find new category with given id %w", err)
		}
	} else {
		newCategory, err = s.Repo.FindCategoryByClassification(tx, "uncategorized", &userID)
		if err != nil {
			return fmt.Errorf("can't find default category %w", err)
		}
	}

	settings, err := s.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// Block before opening
	openAsOf, err := s.AccountService.Repo.GetAccountOpeningAsOf(tx, newAccount.ID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("account has no opening balance; set an opening balance first")
		}
		return err
	}

	newDay := utils.LocalMidnightUTC(req.TxnDate, loc)
	oldDay := utils.LocalMidnightUTC(exTr.TxnDate, loc)
	openDay := utils.LocalMidnightUTC(openAsOf, loc)
	todayDay := utils.LocalMidnightUTC(time.Now(), loc)

	if newDay.Before(openDay) {
		tx.Rollback()
		return fmt.Errorf(
			"transaction date (%s) cannot be before account opening date (%s)",
			newDay.Format("2006-01-02"), openDay.Format("2006-01-02"),
		)
	}

	if newDay.After(todayDay) {
		tx.Rollback()
		return fmt.Errorf(
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
	_, err = s.Repo.UpdateTransaction(tx, tr)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Adjust balances
	signed := func(tt string, amt decimal.Decimal) decimal.Decimal {
		if strings.ToLower(tt) == "expense" {
			return amt.Neg()
		}
		return amt
	}
	dirFor := func(effect decimal.Decimal) string {
		if effect.IsNegative() {
			return "expense"
		}
		return "income"
	}
	reverseDirFor := func(effect decimal.Decimal) string {
		if effect.IsPositive() {
			return "expense"
		}
		return "income"
	}

	oldEffect := signed(exTr.TransactionType, exTr.Amount)
	newEffect := signed(tr.TransactionType, tr.Amount)
	dateChanged := !oldDay.Equal(newDay)

	switch {
	case oldAccount.ID != newAccount.ID || dateChanged:
		// Reverse the old posting on its original day & account
		if !oldEffect.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(
				tx, oldAccount, oldDay, reverseDirFor(oldEffect), oldEffect.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
		// Apply the new posting on the new day & account
		if !newEffect.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(
				tx, newAccount, newDay, dirFor(newEffect), newEffect.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}

	default:
		// Same account & same local day → post only the net delta once
		delta := newEffect.Sub(oldEffect)
		if !delta.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(
				tx, newAccount, newDay, dirFor(delta), delta.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()
	utils.CompareChanges(oldAccount.Name, newAccount.Name, changes, "account")
	utils.CompareChanges(exTr.TransactionType, tr.TransactionType, changes, "type")
	utils.CompareDateChange(&exTr.TxnDate, &tr.TxnDate, changes, "date")
	utils.CompareDecimalChange(&exTr.Amount, &tr.Amount, changes, "amount", 2)
	utils.CompareChanges(exTr.Currency, tr.Currency, changes, "currency")
	utils.CompareChanges(oldCategory.Name, newCategory.Name, changes, "category")
	utils.CompareChanges(utils.SafeString(exTr.Description), utils.SafeString(tr.Description), changes, "description")

	if !changes.IsEmpty() {
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "transaction",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	// Balance change logs
	{
		var delta decimal.Decimal
		if oldAccount.ID == newAccount.ID {
			delta = newEffect.Sub(oldEffect)
		} else {
			delta = newEffect
		}
		if !delta.IsZero() {
			if err := s.AccountService.LogBalanceChange(newAccount, userID, delta); err != nil {
				return err
			}
		}
		if oldAccount.ID != newAccount.ID && !oldEffect.IsZero() {
			if err := s.AccountService.LogBalanceChange(oldAccount, userID, oldEffect.Neg()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TransactionService) UpdateCategory(userID int64, id int64, req *models.CategoryReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	exCat, err := s.Repo.FindCategoryByID(tx, id, &userID, false)
	if err != nil {
		return fmt.Errorf("can't find category with given id %w", err)
	}

	if exCat.IsDefault && (exCat.Classification != req.Classification) {
		return errors.New("can't edit some parts of a default category")
	}

	cat := models.Category{
		ID:             exCat.ID,
		UserID:         &userID,
		Classification: req.Classification,
		DisplayName:    req.DisplayName,
	}

	_, err = s.Repo.UpdateCategory(tx, cat)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	utils.CompareChanges(exCat.DisplayName, cat.DisplayName, changes, "name")
	utils.CompareChanges(exCat.Classification, cat.Classification, changes, "classification")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "category",
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

func (s *TransactionService) DeleteTransaction(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the transaction + relations
	tr, err := s.Repo.FindTransactionByID(tx, id, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find transaction with given id %w", err)
	}

	account, err := s.AccountService.Repo.FindAccountByID(tx, tr.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	// Delete transaction
	if err := s.Repo.DeleteTransaction(tx, tr.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	var category models.Category
	if tr.CategoryID != nil {
		cat, err := s.Repo.FindCategoryByID(tx, *tr.CategoryID, &userID, true)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find category with given id %w", err)
		}
		category = cat
	}

	// Reverse the original cash effect on the account
	signed := func(tt string, amt decimal.Decimal) decimal.Decimal {
		switch strings.ToLower(tt) {
		case "expense":
			return amt.Neg()
		default:
			return amt
		}
	}
	origEffect := signed(tr.TransactionType, tr.Amount)
	inverse := origEffect.Neg()

	if !inverse.IsZero() {
		dir := map[bool]string{true: "expense", false: "income"}[inverse.IsNegative()]
		if err := s.AccountService.UpdateAccountCashBalance(tx, account, tr.TxnDate, dir, inverse.Abs()); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := s.AccountService.FrontfillBalancesForAccount(tx, userID, account.ID, account.Currency, tr.TxnDate); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()

	utils.CompareChanges(account.Name, "", changes, "account")
	utils.CompareChanges(tr.TransactionType, "", changes, "type")
	utils.CompareDateChange(&tr.TxnDate, nil, changes, "date")
	utils.CompareDecimalChange(&tr.Amount, nil, changes, "amount", 2)
	utils.CompareChanges(tr.Currency, "", changes, "currency")
	utils.CompareChanges(utils.SafeString(&category.Name), "", changes, "category")
	utils.CompareChanges(utils.SafeString(tr.Description), "", changes, "description")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
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

	// Dispatch balance change on the affected account activity log
	if !inverse.IsZero() {
		if err := s.AccountService.LogBalanceChange(account, userID, inverse); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteTransfer(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the transfer
	transfer, err := s.Repo.FindTransferByID(tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find transfer with given id %w", err)
	}

	// Load associated transactions
	inflow, err := s.Repo.FindTransactionByID(tx, transfer.TransactionInflowID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}

	outflow, err := s.Repo.FindTransactionByID(tx, transfer.TransactionOutflowID, userID, false)
	if err != nil {
		return fmt.Errorf("can't find outflow transaction with given id %w", err)
	}

	// Load accounts
	fromAcc, err := s.AccountService.Repo.FindAccountByID(tx, outflow.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}
	toAcc, err := s.AccountService.Repo.FindAccountByID(tx, inflow.AccountID, userID, false)
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

	if err := s.AccountService.UpdateBalancesForTransfer(
		tx,
		fromAcc,
		toAcc,
		outflow.TxnDate,
		outflow.Amount.Neg(),
	); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.AccountService.FrontfillBalancesForAccount(tx, userID, fromAcc.ID, fromAcc.Currency, outflow.TxnDate); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.AccountService.FrontfillBalancesForAccount(tx, userID, toAcc.ID, toAcc.Currency, outflow.TxnDate); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transfer
	if err := s.Repo.DeleteTransfer(tx, transfer.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transactions
	if err := s.Repo.DeleteTransaction(tx, inflow.ID, userID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.Repo.DeleteTransaction(tx, outflow.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log synthetic transfer deletion
	changes := utils.InitChanges()
	utils.CompareChanges(fromAcc.Name, "", changes, "from")
	utils.CompareChanges(toAcc.Name, "", changes, "to")
	utils.CompareChanges(transfer.Amount.StringFixed(2), "", changes, "amount")
	utils.CompareChanges(transfer.Currency, "", changes, "currency")
	utils.CompareChanges(utils.SafeString(transfer.Notes), "", changes, "description")

	if !changes.IsEmpty() {
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "delete",
			Category:    "transfer",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	// Log balance changes
	if err := s.AccountService.LogBalanceChange(fromAcc, userID, outflow.Amount); err != nil {
		return err
	}
	if err := s.AccountService.LogBalanceChange(toAcc, userID, inflow.Amount.Neg()); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) DeleteCategory(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	cat, err := s.Repo.FindCategoryByID(tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category with given id: %w", err)
	}

	alreadySoftDeleted := cat.DeletedAt != nil
	var deleteType string

	switch {
	case !alreadySoftDeleted:
		// Archive first
		if err := s.Repo.ArchiveCategory(tx, cat.ID, userID); err != nil {
			tx.Rollback()
			return err
		}
		deleteType = "soft"

	case !cat.IsDefault && alreadySoftDeleted:
		// Non-default category, already archived -> try permanent delete
		cnt, err := s.Repo.CountActiveTransactionsForCategory(tx, userID, cat.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
		if cnt > 0 {
			tx.Rollback()
			return fmt.Errorf("cannot permanently delete category: %d active transactions still reference it", cnt)
		}
		if err := s.Repo.DeleteCategory(tx, cat.ID, userID); err != nil {
			tx.Rollback()
			return err
		}
		deleteType = "hard"
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(deleteType, "", changes, "delete_type")
	utils.CompareChanges(cat.DisplayName, "", changes, "name")
	utils.CompareChanges(cat.Classification, "", changes, "classification")

	if !changes.IsEmpty() {
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
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

func (s *TransactionService) RestoreTransaction(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the transaction
	tr, err := s.Repo.FindTransactionByID(tx, id, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}
	if tr.DeletedAt == nil {
		tx.Rollback()
		return fmt.Errorf("transaction is not deleted")
	}

	// Load account
	acc, err := s.AccountService.Repo.FindAccountByID(tx, tr.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account for transaction %w", err)
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
		if err := s.AccountService.UpdateAccountCashBalance(tx, acc, tr.TxnDate, dir, origEffect.Abs()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Unmark as soft deleted
	if err := s.Repo.RestoreTransaction(tx, tr.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", acc.Name, changes, "account")
	utils.CompareChanges("", tr.Amount.StringFixed(2), changes, "amount")
	utils.CompareChanges("", tr.Currency, changes, "currency")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "restore",
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	// Log balance changes
	if !origEffect.IsZero() {
		if err := s.AccountService.LogBalanceChange(acc, userID, origEffect); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) RestoreCategory(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the record
	cat, err := s.Repo.FindCategoryByID(tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing category with given id %w", err)
	}
	if cat.DeletedAt == nil {
		tx.Rollback()
		return fmt.Errorf("category is not deleted")
	}

	// Unmark as soft deleted
	if err := s.Repo.RestoreCategory(tx, cat.ID, &userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", cat.DisplayName, changes, "name")
	utils.CompareChanges("", cat.Classification, changes, "classification")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
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

func (s *TransactionService) RestoreCategoryName(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the record
	cat, err := s.Repo.FindCategoryByID(tx, id, &userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing category with given id %w", err)
	}

	changes := utils.InitChanges()
	utils.CompareChanges(utils.NormalizeName(cat.DisplayName), cat.Name, changes, "name")

	if err := s.Repo.RestoreCategoryName(tx, cat.ID, &userID, cat.Name); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
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

func (s *TransactionService) FetchTransactionTemplatesPaginated(userID int64, p utils.PaginationParams) ([]models.TransactionTemplate, *utils.Paginator, error) {

	totalRecords, err := s.Repo.CountTransactionTemplates(userID, false)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindTransactionTemplates(userID, offset, p.RowsPerPage)
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

func (s *TransactionService) FetchTransactionTemplateByID(userID int64, id int64) (*models.TransactionTemplate, error) {

	record, err := s.Repo.FindTransactionTemplateByID(nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) InsertTransactionTemplate(userID int64, req *models.TransactionTemplateReq) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	account, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	category, err := s.Repo.FindCategoryByID(tx, req.CategoryID, &userID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category with given id %w", err)
	}

	firstRun := time.Date(
		req.NextRunAt.Year(), req.NextRunAt.Month(), req.NextRunAt.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	firstValidDay := time.Now().In(time.UTC).Truncate(24 * time.Hour)

	if firstRun.Before(firstValidDay) {
		tx.Rollback()
		return fmt.Errorf(
			"first itteration of template cannot be executed in the same day (%s)",
			firstValidDay.Format("2006-01-02"),
		)
	}

	if req.MaxRuns != nil {
		if *req.MaxRuns < 0 || *req.MaxRuns > 99999 {
			tx.Rollback()
			return fmt.Errorf("max runs out of bounds %w", err)
		}
	}

	var endDate *time.Time
	if req.EndDate != nil {
		e := time.Date(
			req.EndDate.Year(), req.EndDate.Month(), req.EndDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)
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

	_, err = s.Repo.InsertTransactionTemplate(tx, &tp)
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

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
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

func (s *TransactionService) UpdateTransactionTemplate(userID, id int64, req *models.TransactionTemplateReq) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	changes := utils.InitChanges()

	// Load existing transaction template
	exTp, err := s.Repo.FindTransactionTemplateByID(tx, id, userID)
	if err != nil {
		return fmt.Errorf("can't find transaction template with given id %w", err)
	}

	nextRun := time.Date(
		req.NextRunAt.Year(), req.NextRunAt.Month(), req.NextRunAt.Day(),
		0, 0, 0, 0, time.UTC,
	)
	firstValidDay := time.Now().In(time.UTC).Truncate(24 * time.Hour)
	if nextRun.Before(firstValidDay) {
		tx.Rollback()
		return fmt.Errorf("next run cannot be today or earlier (%s)", firstValidDay.Format("2006-01-02"))
	}

	if req.MaxRuns != nil {
		if *req.MaxRuns < 0 || *req.MaxRuns > 99999 {
			tx.Rollback()
			return fmt.Errorf("max runs out of bounds %w", err)
		}
	}

	var endDate *time.Time
	if req.EndDate != nil {
		e := time.Date(
			req.EndDate.Year(), req.EndDate.Month(), req.EndDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)
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

	_, err = s.Repo.UpdateTransactionTemplate(tx, tp, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	exAmountString := exTp.Amount.StringFixed(2)
	amountString := tp.Amount.StringFixed(2)
	exNextRunStr := tp.NextRunAt.UTC().Format(time.RFC3339)
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

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "update",
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

func (s *TransactionService) ToggleTransactionTemplateActiveState(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load record to confirm it exists
	exTp, err := s.Repo.FindTransactionTemplateByID(tx, id, userID)
	if err != nil {
		return fmt.Errorf("can't find transaction template with given id %w", err)
	}

	tp := models.TransactionTemplate{
		ID:       exTp.ID,
		UserID:   userID,
		IsActive: !exTp.IsActive,
	}

	changes := utils.InitChanges()
	utils.CompareChanges(strconv.FormatBool(exTp.IsActive), strconv.FormatBool(tp.IsActive), changes, "is_active")

	_, err = s.Repo.UpdateTransactionTemplate(tx, tp, true)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
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

func (s *TransactionService) DeleteTransactionTemplate(userID int64, id int64) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Confirm existence
	tp, err := s.Repo.FindTransactionTemplateByID(tx, id, userID)
	if err != nil {
		return fmt.Errorf("can't find transaction template with given id %w", err)
	}

	err = s.Repo.DeleteTransactionTemplate(tx, tp.ID)
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

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
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

func (s *TransactionService) GetTransactionTemplateCount(userID int64) (int64, error) {
	return s.Repo.CountTransactionTemplates(userID, true)
}

func (s *TransactionService) FetchAllCategoryGroups(userID int64) ([]models.CategoryGroup, error) {

	categories, err := s.Repo.FindAllCategoryGroups(nil, userID)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *TransactionService) FetchCategoryGroupByID(userID int64, id int64) (*models.CategoryGroup, error) {

	record, err := s.Repo.FindCategoryGroupByID(nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) InsertCategoryGroup(userID int64, req *models.CategoryGroupReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
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

	groupingID, err := s.Repo.InsertCategoryGroup(tx, &rec)
	if err != nil {
		tx.Rollback()
		return err
	}

	categoryIDs, ok := req.SelectedCategories.([]interface{})
	if !ok || len(categoryIDs) == 0 {
		tx.Rollback()
		return fmt.Errorf("invalid or empty selected_categories")
	}

	for _, idVal := range categoryIDs {
		categoryID, _ := strconv.ParseInt(fmt.Sprint(idVal), 10, 64)

		// Validate category exists
		_, err := s.Repo.FindCategoryByID(tx, categoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to validate category %d: %w", categoryID, err)
		}

		// Create the m:m relation
		if err := s.Repo.InsertCategoryGroupMember(tx, groupingID, categoryID); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to link category %d: %w", categoryID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Log transfer (one event)
	changes := utils.InitChanges()
	utils.CompareChanges("", rec.Name, changes, "name")
	utils.CompareChanges("", rec.Classification, changes, "classification")
	utils.CompareChanges("", fmt.Sprintf("%d categories", len(categoryIDs)), changes, "categories_count")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "category_group",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) UpdateCategoryGroup(userID int64, id int64, req *models.CategoryGroupReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Get existing record
	exGroup, err := s.Repo.FindCategoryGroupByID(tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category group with given id: %w", err)
	}

	rec := models.CategoryGroup{
		ID:             id,
		UserID:         &userID,
		Classification: req.Classification,
		Name:           req.Name,
		Description:    req.Description,
	}

	_, err = s.Repo.UpdateCategoryGroup(tx, rec)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete existing category relations
	if err := s.Repo.DeleteCategoryGroupMembers(tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear existing categories: %w", err)
	}

	// Add new category relations
	categoryIDs, ok := req.SelectedCategories.([]interface{})
	if !ok || len(categoryIDs) == 0 {
		tx.Rollback()
		return fmt.Errorf("invalid or empty selected_categories")
	}

	for _, idVal := range categoryIDs {
		categoryID, _ := strconv.ParseInt(fmt.Sprint(idVal), 10, 64)

		// Validate category exists
		_, err := s.Repo.FindCategoryByID(tx, categoryID, &userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to validate category %d: %w", categoryID, err)
		}

		// Create the m:m relation
		if err := s.Repo.InsertCategoryGroupMember(tx, id, categoryID); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to link category %d: %w", categoryID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(exGroup.Name, rec.Name, changes, "name")
	utils.CompareChanges(exGroup.Classification, rec.Classification, changes, "classification")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "category_group",
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

func (s *TransactionService) DeleteCategoryGroup(userID int64, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	group, err := s.Repo.FindCategoryGroupByID(tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find category group with given id: %w", err)
	}

	// Delete all category relations first
	if err := s.Repo.DeleteCategoryGroupMembers(tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete category relations: %w", err)
	}

	// Delete the group itself
	if err := s.Repo.DeleteCategoryGroup(tx, id, userID); err != nil {
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
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
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
