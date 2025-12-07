package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountServiceInterface interface {
	FetchAccountsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeInactive bool, classification string) ([]models.Account, *utils.Paginator, error)
	FetchLatestBalance(ctx context.Context, accID, userID int64) (*models.Balance, error)
	FetchAccountByID(ctx context.Context, userID int64, id int64, initialBalance bool) (*models.Account, error)
	FetchAccountByName(ctx context.Context, userID int64, name string) (*models.Account, error)
	FetchAllAccounts(ctx context.Context, userID int64, includeInactive bool) ([]models.Account, error)
	FetchAllAccountTypes(ctx context.Context) ([]models.AccountType, error)
	FetchAccountsBySubtype(ctx context.Context, userID int64, subtype string) ([]models.Account, error)
	FetchAccountsByType(ctx context.Context, userID int64, t string) ([]models.Account, error)
	InsertAccount(ctx context.Context, userID int64, req *models.AccountReq) (int64, error)
	UpdateAccount(ctx context.Context, userID int64, id int64, req *models.AccountReq) (int64, error)
	ToggleAccountActiveState(ctx context.Context, userID int64, id int64) error
	CloseAccount(ctx context.Context, userID int64, id int64) error
	UpdateAccountCashBalance(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, transactionType string, amount decimal.Decimal) error
	UpdateBalancesForTransfer(ctx context.Context, tx *gorm.DB, fromAcc, toAcc *models.Account, when time.Time, amount decimal.Decimal) error
	BackfillBalancesForUser(ctx context.Context, userID int64, from, to string) error
	FrontfillBalancesForAccount(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from time.Time) error
	UpdateDailyCashNoSnapshot(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, txnType string, amt decimal.Decimal) error
	SaveAccountProjection(ctx context.Context, id, userID int64, req *models.AccountProjectionReq) error
	RevertAccountProjection(ctx context.Context, id, userID int64) error
}

type AccountService struct {
	repo          repositories.AccountRepositoryInterface
	txnRepo       repositories.TransactionRepositoryInterface
	settingsRepo  repositories.SettingsRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobs.JobDispatcher
}

func NewAccountService(
	repo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobs.JobDispatcher,
) *AccountService {
	return &AccountService{
		repo:          repo,
		txnRepo:       txnRepo,
		settingsRepo:  settingsRepo,
		jobDispatcher: jobDispatcher,
		loggingRepo:   loggingRepo,
	}
}

var _ AccountServiceInterface = (*AccountService)(nil)

func (s *AccountService) LogBalanceChange(ctx context.Context, account *models.Account, userID int64, change decimal.Decimal) error {
	newBalance, err := s.repo.FindBalanceForAccountID(ctx, nil, account.ID)
	if err != nil {
		return err
	}

	endBalance := newBalance.EndBalance
	startBalance := endBalance.Sub(change)

	changes := utils.InitChanges()
	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", change.StringFixed(2), changes, "change")
	utils.CompareChanges("", startBalance.StringFixed(2), changes, "start_balance")
	utils.CompareChanges("", endBalance.StringFixed(2), changes, "end_balance")
	utils.CompareChanges("", account.Currency, changes, "currency")

	return s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "update",
		Category:    "balance",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
}

func (s *AccountService) FetchAccountsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeInactive bool, classification string) ([]models.Account, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountAccounts(ctx, nil, userID, p.Filters, includeInactive, &classification)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage
	records, err := s.repo.FindAccounts(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeInactive, &classification)
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

func (s *AccountService) FetchLatestBalance(ctx context.Context, accID, userID int64) (*models.Balance, error) {

	record, err := s.repo.FindLatestBalance(ctx, nil, accID, userID)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAccountByID(ctx context.Context, userID int64, id int64, initialBalance bool) (*models.Account, error) {

	if initialBalance {
		record, err := s.repo.FindAccountByIDWithInitialBalance(ctx, nil, id, userID)
		if err != nil {
			return nil, err
		}
		return record, nil
	}
	record, err := s.repo.FindAccountByID(ctx, nil, id, userID, true)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAccountByName(ctx context.Context, userID int64, name string) (*models.Account, error) {

	record, err := s.repo.FindAccountByName(ctx, nil, userID, name)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAllAccounts(ctx context.Context, userID int64, includeInactive bool) ([]models.Account, error) {
	return s.repo.FindAllAccounts(ctx, nil, userID, includeInactive)
}

func (s *AccountService) FetchAllAccountTypes(ctx context.Context) ([]models.AccountType, error) {
	return s.repo.FindAllAccountTypes(ctx, nil, nil)
}

func (s *AccountService) FetchAccountsBySubtype(ctx context.Context, userID int64, subtype string) ([]models.Account, error) {
	return s.repo.FindAccountsBySubtype(ctx, nil, userID, subtype, true)
}

func (s *AccountService) FetchAccountsByType(ctx context.Context, userID int64, t string) ([]models.Account, error) {
	return s.repo.FetchAccountsByType(ctx, nil, userID, t, true)
}

func (s *AccountService) InsertAccount(ctx context.Context, userID int64, req *models.AccountReq) (int64, error) {

	changes := utils.InitChanges()

	if req.Classification == "asset" && req.Balance.LessThan(decimal.NewFromInt(0)) {
		return 0, errors.New("provided initial balance cannot be negative")
	}

	accCount, err := s.repo.CountAccounts(ctx, nil, userID, nil, false, nil)
	if err != nil {
		return 0, err
	}

	maxAcc, err := s.settingsRepo.FetchMaxAccountsForUser(ctx, nil)
	if err != nil {
		return 0, err
	}

	if accCount >= maxAcc {
		return 0, fmt.Errorf("you can only have %d active accounts", maxAcc)
	}

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

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	openedAt := req.OpenedAt
	if openedAt.IsZero() {
		openedAt = time.Now().UTC()
	}
	openedDay := utils.LocalMidnightUTC(openedAt, loc)

	accType, err := s.repo.FindAccountTypeByID(ctx, tx, req.AccountTypeID)
	if err != nil {
		return 0, fmt.Errorf("can't find account_type for given id %w", err)
	}

	account := &models.Account{
		Name:              req.Name,
		Currency:          models.DefaultCurrency,
		AccountTypeID:     accType.ID,
		UserID:            userID,
		OpenedAt:          openedDay,
		BalanceProjection: "fixed",
	}

	balanceAmountString := req.Balance.StringFixed(2)
	dateStr := account.OpenedAt.UTC().Format(time.RFC3339)

	utils.CompareChanges("", account.Name, changes, "name")
	utils.CompareChanges("", accType.Type, changes, "account_type")
	utils.CompareChanges("", accType.Subtype, changes, "account_subtype")
	utils.CompareChanges("", account.Currency, changes, "currency")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")
	utils.CompareChanges("", dateStr, changes, "opened_at")

	accountID, err := s.repo.InsertAccount(ctx, tx, account)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	amount := req.Balance.Round(4)

	if accType.Classification == "liability" {
		amount = amount.Neg()
	}

	asOf := openedDay

	balance := &models.Balance{
		AccountID:    accountID,
		Currency:     models.DefaultCurrency,
		StartBalance: amount,
		AsOf:         asOf,
	}

	_, err = s.repo.InsertBalance(ctx, tx, balance)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// seed snapshots from opened day to today
	if err := s.repo.UpsertSnapshotsFromBalances(
		ctx,
		tx,
		userID,
		accountID,
		models.DefaultCurrency,
		asOf,
		time.Now().UTC().Truncate(24*time.Hour),
	); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "account",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return accountID, nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, userID int64, id int64, req *models.AccountReq) (int64, error) {

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

	// Load record
	exAcc, err := s.repo.FindAccountByIDWithInitialBalance(ctx, tx, id, userID)
	if err != nil {
		return 0, fmt.Errorf("can't find account with given id %w", err)
	}

	if !exAcc.IsActive {
		return 0, errors.New("can't update non-active account")
	}

	// Load existing relations for comparison
	exAccType, err := s.repo.FindAccountTypeByID(ctx, tx, exAcc.AccountTypeID)
	if err != nil {
		return 0, fmt.Errorf("can't find account type with given id %w", err)
	}

	// Resolve new relations  from req
	newAccType, err := s.repo.FindAccountTypeByID(ctx, tx, req.AccountTypeID)
	if err != nil {
		return 0, fmt.Errorf("can't find account type with given id %w", err)
	}

	// Handle OpenedAt change
	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	newOpenedAt := req.OpenedAt
	if newOpenedAt.IsZero() {
		newOpenedAt = exAcc.OpenedAt
	} else {
		newOpenedAt = utils.LocalMidnightUTC(newOpenedAt, loc)
	}

	if !newOpenedAt.Equal(exAcc.OpenedAt) {
		// Check if any transactions exist for this account
		earliestTxnDate, err := s.repo.FindEarliestTransactionDate(ctx, tx, id)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to check transaction dates: %w", err)
		}

		if earliestTxnDate != nil {
			// validate the new date is before earliest transaction
			if !newOpenedAt.Before(*earliestTxnDate) {
				tx.Rollback()
				return 0, fmt.Errorf("opened date must be before the earliest transaction date (%s)",
					earliestTxnDate.Format("2006-01-02"))
			}
		}

		initialBalance := exAcc.Balance.StartBalance

		// Delete all existing snapshots for this account
		err = s.repo.DeleteAccountSnapshots(ctx, tx, id)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to delete existing snapshots: %w", err)
		}

		// Create new initial balance with the true initial amount at the new date
		newInitialBalance := &models.Balance{
			AccountID:    id,
			Currency:     models.DefaultCurrency,
			StartBalance: initialBalance,
			AsOf:         newOpenedAt,
		}

		_, err = s.repo.InsertBalance(ctx, tx, newInitialBalance)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to create new initial balance: %w", err)
		}

		if err := s.repo.FrontfillBalances(ctx, tx, id, models.DefaultCurrency, newOpenedAt); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to rebuild balances from transactions: %w", err)
		}

		// Re-seed snapshots from the new opened date to today
		if err := s.repo.UpsertSnapshotsFromBalances(
			ctx,
			tx,
			userID,
			id,
			models.DefaultCurrency,
			newOpenedAt,
			time.Now().UTC().Truncate(24*time.Hour),
		); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to update snapshots: %w", err)
		}
	}

	acc := &models.Account{
		ID:            id,
		Name:          req.Name,
		Currency:      models.DefaultCurrency,
		AccountTypeID: newAccType.ID,
		IsActive:      exAcc.IsActive,
		UserID:        userID,
		OpenedAt:      newOpenedAt,
	}

	changes := utils.InitChanges()

	utils.CompareChanges(exAcc.Name, acc.Name, changes, "name")
	utils.CompareChanges(exAccType.Type, newAccType.Type, changes, "account_type")
	utils.CompareChanges(exAccType.Subtype, newAccType.Subtype, changes, "account_subtype")
	utils.CompareChanges(exAcc.Currency, acc.Currency, changes, "currency")

	// Compare opened_at dates
	exDateStr := exAcc.OpenedAt.UTC().Format(time.RFC3339)
	newDateStr := newOpenedAt.UTC().Format(time.RFC3339)
	utils.CompareChanges(exDateStr, newDateStr, changes, "opened_at")

	var delta decimal.Decimal

	if req.Balance != nil {

		desired, err := decimal.NewFromString(req.Balance.String())
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("invalid balance value: %w", err)
		}

		latestBalance, err := s.repo.FindLatestBalance(ctx, tx, exAcc.ID, userID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		// Match sign conventions
		isLiability := strings.EqualFold(newAccType.Type, "liability")

		delta = desired.Sub(latestBalance.EndBalance)
		signed := delta
		if isLiability {
			signed = delta.Neg()
		}

		if !signed.IsZero() {
			txnType := "income"
			amount := signed
			if signed.IsNegative() {
				txnType = "expense"
				amount = signed.Neg()
			}

			desc := "Manual adjustment"

			category, err := s.txnRepo.FindCategoryByClassification(ctx, tx, "adjustment", &userID)
			if err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("can't find adjustment category: %w", err)
			}

			txn := &models.Transaction{
				UserID:          userID,
				AccountID:       exAcc.ID,
				CategoryID:      &category.ID,
				TransactionType: txnType,
				Amount:          amount,
				Currency:        exAcc.Currency,
				TxnDate:         time.Now().UTC(),
				Description:     &desc,
				IsAdjustment:    true,
			}

			if _, err := s.txnRepo.InsertTransaction(ctx, tx, txn); err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("failed to post adjustment transaction: %w", err)
			}

			err = s.UpdateAccountCashBalance(ctx, tx, acc, txn.TxnDate, txnType, amount)
			if err != nil {
				tx.Rollback()
				return 0, err
			}

		}
	}

	accID, err := s.repo.UpdateAccount(ctx, tx, acc)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	// balance log (with the new end_balance)
	if req.Balance != nil {
		accForLog := &models.Account{ID: exAcc.ID, Name: acc.Name, Currency: exAcc.Currency}
		if err := s.LogBalanceChange(ctx, accForLog, userID, delta); err != nil {
			fmt.Println("Balance change logging failed")
		}
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "account",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return accID, nil
}

func (s *AccountService) ToggleAccountActiveState(ctx context.Context, userID int64, id int64) error {

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
	exAcc, err := s.repo.FindAccountByID(ctx, tx, id, userID, false)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	accCount, err := s.repo.CountAccounts(ctx, tx, userID, nil, false, nil)
	if err != nil {
		return err
	}

	maxAcc, err := s.settingsRepo.FetchMaxAccountsForUser(ctx, nil)
	if err != nil {
		return err
	}

	if !exAcc.IsActive && accCount >= maxAcc {
		return fmt.Errorf("you can only have %d active accounts", maxAcc)
	}

	acc := &models.Account{
		ID:       id,
		UserID:   userID,
		IsActive: !exAcc.IsActive,
	}

	changes := utils.InitChanges()
	utils.CompareChanges(strconv.FormatBool(exAcc.IsActive), strconv.FormatBool(acc.IsActive), changes, "is_active")

	_, err = s.repo.UpdateAccount(ctx, tx, acc)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "account",
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

func (s *AccountService) CloseAccount(ctx context.Context, userID int64, id int64) error {

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

	// Load the account
	acc, err := s.repo.FindAccountByID(ctx, tx, id, userID, false)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	// Close it
	if err := s.repo.CloseAccount(ctx, tx, acc.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Materialize a real snapshot for today so charts don’t copy yesterday’s value
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Upsert for the just-closed account
	_ = s.repo.UpsertSnapshotsFromBalances(ctx, tx, userID, acc.ID, acc.Currency, today, today)

	// Upsert for all still-open accounts for today (so the view has a “today” row)
	openAccs, err := s.repo.FindAllAccounts(ctx, tx, userID, false)
	if err == nil {
		for _, a := range openAccs {
			_ = s.repo.UpsertSnapshotsFromBalances(ctx, tx, userID, a.ID, a.Currency, today, today)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	utils.CompareChanges(acc.Name, "", changes, "account")
	utils.CompareChanges(acc.AccountType.Type, "", changes, "type")
	utils.CompareChanges(acc.AccountType.Subtype, "", changes, "sub_type")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "close",
			Category:    "account",
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

func (s *AccountService) UpdateAccountCashBalance(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, transactionType string, amount decimal.Decimal) error {
	// ensure daily balance row exists for asOf
	if err := s.repo.EnsureDailyBalanceRow(ctx, tx, acc.ID, asOf, acc.Currency); err != nil {
		return err
	}

	amount = amount.Round(4)

	// increment the correct field on balances(as_of)
	switch strings.ToLower(transactionType) {
	case "expense":
		// expense decreases cash => goes to cash_outflows
		if err := s.repo.AddToDailyBalance(ctx, tx, acc.ID, asOf, "cash_outflows", amount); err != nil {
			return err
		}
	default:
		// income increases cash => goes to cash_inflows
		if err := s.repo.AddToDailyBalance(ctx, tx, acc.ID, asOf, "cash_inflows", amount); err != nil {
			return err
		}
	}

	if err := s.repo.UpsertSnapshotsFromBalances(
		ctx,
		tx,
		acc.UserID,
		acc.ID,
		acc.Currency,
		asOf.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	); err != nil {
		return err
	}

	return nil
}

func (s *AccountService) UpdateBalancesForTransfer(ctx context.Context, tx *gorm.DB, fromAcc, toAcc *models.Account, when time.Time, amount decimal.Decimal) error {
	if err := s.UpdateAccountCashBalance(ctx, tx, fromAcc, when, "expense", amount); err != nil {
		return err
	}

	if err := s.UpdateAccountCashBalance(ctx, tx, toAcc, when, "income", amount); err != nil {
		return err
	}

	return nil
}

func (s *AccountService) BackfillBalancesForUser(ctx context.Context, userID int64, from, to string) error {

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

	accounts, err := s.repo.FindAllAccounts(ctx, tx, userID, true)
	if err != nil {
		tx.Rollback()
		return err
	}
	if len(accounts) == 0 {
		return tx.Commit().Error
	}

	dfrom, dto, err := s.resolveUserDateRange(ctx, tx, userID, from, to)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, acc := range accounts {
		if err := s.backfillAccountRange(ctx, tx, &acc, dfrom, dto); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *AccountService) resolveUserDateRange(ctx context.Context, tx *gorm.DB, userID int64, from, to string) (time.Time, time.Time, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	var dfrom time.Time
	var dto time.Time
	var err error

	if strings.TrimSpace(to) == "" {
		dto = today
	} else {
		dto, err = time.Parse("2006-01-02", to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date: %w", err)
		}
	}

	if strings.TrimSpace(from) != "" {
		dfrom, err = time.Parse("2006-01-02", from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date: %w", err)
		}
	} else {
		// default from = min(first balance as_of, first txn date, today)
		fb, err := s.repo.GetUserFirstBalanceDate(ctx, tx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		ft, err := s.repo.GetUserFirstTxnDate(ctx, tx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		dfrom = today
		if !fb.IsZero() && fb.Before(dfrom) {
			dfrom = fb
		}
		if !ft.IsZero() && ft.Before(dfrom) {
			dfrom = ft
		}
	}

	if dfrom.After(dto) {
		// clamp: at least a single day
		dfrom = dto
	}
	return dfrom, dto, nil
}

func (s *AccountService) backfillAccountRange(ctx context.Context, tx *gorm.DB, acc *models.Account, dfrom, dto time.Time) error {
	return s.repo.UpsertSnapshotsFromBalances(
		ctx,
		tx,
		acc.UserID,
		acc.ID,
		acc.Currency,
		dfrom,
		dto,
	)
}

func (s *AccountService) FrontfillBalancesForAccount(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from time.Time) error {

	from = from.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.repo.FrontfillBalances(ctx, tx, accountID, currency, from); err != nil {
		tx.Rollback()
		return err
	}

	// recompute snapshots
	if err := s.repo.UpsertSnapshotsFromBalances(ctx, tx, userID, accountID, currency, from, today); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (s *AccountService) UpdateDailyCashNoSnapshot(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, txnType string, amt decimal.Decimal) error {
	if err := s.repo.EnsureDailyBalanceRow(ctx, tx, acc.ID, asOf, acc.Currency); err != nil {
		return err
	}
	amt = amt.Round(4)
	if strings.ToLower(txnType) == "expense" {
		return s.repo.AddToDailyBalance(ctx, tx, acc.ID, asOf, "cash_outflows", amt)
	}
	return s.repo.AddToDailyBalance(ctx, tx, acc.ID, asOf, "cash_inflows", amt)
}

func (s *AccountService) SaveAccountProjection(ctx context.Context, id, userID int64, req *models.AccountProjectionReq) error {

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

	exAcc, err := s.repo.FindAccountByID(ctx, tx, id, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	acc := &models.Account{
		ID:                id,
		UserID:            userID,
		ExpectedBalance:   req.ExpectedBalance,
		BalanceProjection: req.BalanceProjection,
	}

	changes := utils.InitChanges()

	utils.CompareChanges(exAcc.Name, acc.Name, changes, "name")
	utils.CompareChanges(exAcc.BalanceProjection, acc.BalanceProjection, changes, "balance_projection")
	utils.CompareChanges("", "save", changes, "action")
	utils.CompareChanges(exAcc.ExpectedBalance.String(), acc.ExpectedBalance.String(), changes, "expected_balance")

	_, err = s.repo.UpdateAccountProjection(ctx, tx, acc)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "account_projection",
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

func (s *AccountService) RevertAccountProjection(ctx context.Context, id, userID int64) error {

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

	exAcc, err := s.repo.FindAccountByID(ctx, tx, id, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	acc := &models.Account{
		ID:                id,
		UserID:            userID,
		ExpectedBalance:   decimal.NewFromInt(0),
		BalanceProjection: "fixed",
	}

	changes := utils.InitChanges()

	utils.CompareChanges(exAcc.Name, acc.Name, changes, "name")
	utils.CompareChanges(exAcc.BalanceProjection, acc.BalanceProjection, changes, "balance_projection")
	utils.CompareChanges("", "revert", changes, "action")
	utils.CompareChanges(exAcc.ExpectedBalance.String(), acc.ExpectedBalance.String(), changes, "expected_balance")

	_, err = s.repo.UpdateAccountProjection(ctx, tx, acc)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "account_projection",
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
