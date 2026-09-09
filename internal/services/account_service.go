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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrAccountNotEmpty = errors.New("account must have a zero balance before it can be closed")

type AccountServiceInterface interface {
	FetchAccountsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, includeInactive bool, classification string) ([]models.Account, *utils.Paginator, error)
	FetchLatestBalance(ctx context.Context, accID, userID int64) (*models.AccountBalance, error)
	FetchAccountByID(ctx context.Context, userID int64, id int64) (*models.Account, error)
	FetchAccountWithOpening(ctx context.Context, userID int64, id int64) (*models.AccountWithOpening, error)
	FetchAccountByName(ctx context.Context, userID int64, name string) (*models.Account, error)
	FetchAllAccounts(ctx context.Context, userID int64, includeInactive bool, options ...bool) ([]models.Account, error)
	FetchAllAccountTypes(ctx context.Context) ([]models.AccountType, error)
	FetchAccountsBySubtype(ctx context.Context, userID int64, subtype string) ([]models.Account, error)
	FetchAccountsByType(ctx context.Context, userID int64, t string) ([]models.Account, error)
	InsertAccount(ctx context.Context, userID int64, req *models.AccountReq) (int64, error)
	UpdateAccount(ctx context.Context, userID int64, id int64, req *models.AccountReq) (int64, error)
	ToggleAccountActiveState(ctx context.Context, userID int64, id int64) error
	CloseAccount(ctx context.Context, userID int64, id int64) error
	FetchAccountsForUser(ctx context.Context, userID int64) ([]models.AccountLookup, error)
	PurgeAccount(ctx context.Context, actorID, accountID int64) error
	UpdateAccountCashBalance(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time) error
	BackfillBalancesForUser(ctx context.Context, userID int64, from, to string) error
	UpdateDailyCashNoSnapshot(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, txnType string, amt decimal.Decimal) error
	SaveAccountProjection(ctx context.Context, id, userID int64, req *models.AccountProjectionReq) error
	RevertAccountProjection(ctx context.Context, id, userID int64) error
	FetchAccountsWithDefaults(ctx context.Context, userID int64) ([]models.Account, error)
	FetchAccountTypesWithoutDefaults(ctx context.Context, userID int64) ([]models.AccountType, error)
	SetDefaultAccount(ctx context.Context, userID, accountID int64) error
	UnsetDefaultAccount(ctx context.Context, userID, accountID int64) error
	UpdateSnapshotMarketValues(ctx context.Context, userID int64, from time.Time) error
	SyncForUser(ctx context.Context, userID int64) error
	RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error
	GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error)
	SyncAssetPnL(ctx context.Context, userID, assetID int64) error
	SyncAccountPnL(ctx context.Context, userID, accountID int64) error
	QueueAccountMerge(ctx context.Context, userID, sourceID, destinationID int64) error
	MergeAccount(ctx context.Context, userID, sourceID, destinationID int64) error
}

type AccountService struct {
	repo             repositories.AccountRepositoryInterface
	balanceRepo      repositories.BalanceRepositoryInterface
	txnRepo          repositories.TransactionRepositoryInterface
	settingsRepo     repositories.SettingsRepositoryInterface
	savingsRepo      repositories.SavingsRepositoryInterface
	investmentRepo   repositories.InvestmentRepositoryInterface
	jobDispatcher    jobqueue.Dispatcher
	priceFetchClient finance.PriceFetcher
	logger           *zap.Logger
}

func NewAccountService(
	logger *zap.Logger,
	repo *repositories.AccountRepository,
	balanceRepo *repositories.BalanceRepository,
	txnRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
	savingsRepo *repositories.SavingsRepository,
	investmentRepo *repositories.InvestmentRepository,
	jobDispatcher jobqueue.Dispatcher,
	priceFetchClient finance.PriceFetcher,
) *AccountService {
	return &AccountService{
		repo:             repo,
		balanceRepo:      balanceRepo,
		txnRepo:          txnRepo,
		settingsRepo:     settingsRepo,
		savingsRepo:      savingsRepo,
		investmentRepo:   investmentRepo,
		jobDispatcher:    jobDispatcher,
		priceFetchClient: priceFetchClient,
		logger:           logger,
	}
}

var _ AccountServiceInterface = (*AccountService)(nil)

func (s *AccountService) LogBalanceChange(ctx context.Context, account *models.Account, userID int64, change decimal.Decimal) error {
	endBalance, err := s.balanceRepo.GetBalance(ctx, nil, account.ID)
	if err != nil {
		return err
	}

	startBalance := endBalance.Sub(change)

	changes := utils.InitChanges()
	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", change.StringFixed(2), changes, "change")
	utils.CompareChanges("", startBalance.StringFixed(2), changes, "start_balance")
	utils.CompareChanges("", endBalance.StringFixed(2), changes, "end_balance")
	utils.CompareChanges("", account.Currency, changes, "currency")
	changes.Stamp("id", strconv.FormatInt(account.ID, 10))

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *AccountService) FetchLatestBalance(ctx context.Context, accID, userID int64) (*models.AccountBalance, error) {

	record, err := s.balanceRepo.FindAccountBalance(ctx, nil, accID, userID)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAccountByID(ctx context.Context, userID int64, id int64) (*models.Account, error) {

	record, err := s.repo.FindAccountByID(ctx, nil, id, userID, true)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAccountWithOpening(ctx context.Context, userID int64, id int64) (*models.AccountWithOpening, error) {

	record, opening, err := s.repo.FindAccountByIDWithOpening(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &models.AccountWithOpening{Account: *record, StartBalance: opening}, nil
}

func (s *AccountService) FetchAccountByName(ctx context.Context, userID int64, name string) (*models.Account, error) {

	record, err := s.repo.FindAccountByName(ctx, nil, userID, name)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *AccountService) FetchAllAccounts(ctx context.Context, userID int64, includeInactive bool, options ...bool) ([]models.Account, error) {
	includeAccountTypes := false
	if len(options) > 0 {
		includeAccountTypes = options[0]
	}
	return s.repo.FindAllAccounts(ctx, nil, userID, includeInactive, includeAccountTypes)
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

func (s *AccountService) FetchAccountTypesByClassification(ctx context.Context, c string) ([]models.AccountType, error) {
	return s.repo.FindAccountTypeClassification(ctx, nil, c)
}

func (s *AccountService) InsertAccount(ctx context.Context, userID int64, req *models.AccountReq) (int64, error) {

	changes := utils.InitChanges()

	if req.Classification == "asset" {
		if req.CreditLimit != nil {
			if req.Balance.LessThan(req.CreditLimit.Neg()) {
				return 0, errors.New("provided initial balance cannot be below credit limit")
			}
		} else if req.Balance.LessThan(decimal.NewFromInt(0)) {
			return 0, errors.New("provided initial balance cannot be negative")
		}
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
		Currency:          settings.DefaultCurrency,
		AccountTypeID:     accType.ID,
		UserID:            userID,
		OpenedAt:          openedDay,
		BalanceProjection: "fixed",
	}
	if accType.Classification != "liability" {
		account.CreditLimit = req.CreditLimit
	}

	balanceAmountString := req.Balance.StringFixed(2)
	dateStr := account.OpenedAt.UTC().Format(time.RFC3339)

	accountID, err := s.repo.InsertAccount(ctx, tx, account)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	utils.CompareChanges("", strconv.FormatInt(accountID, 10), changes, "id")
	utils.CompareChanges("", account.Name, changes, "name")
	utils.CompareChanges("", accType.Type, changes, "account_type")
	utils.CompareChanges("", accType.Subtype, changes, "account_subtype")
	utils.CompareChanges("", account.Currency, changes, "currency")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")
	utils.CompareChanges("", dateStr, changes, "opened_at")

	amount := req.Balance.Round(4)

	if accType.Classification == "liability" {
		amount = amount.Neg()
	}

	// The opening row is user editable, and the edit form needs a category on it.
	openingCategory, err := s.txnRepo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find uncategorized category: %w", err)
	}

	openingTxn := models.NewOpeningTransaction(userID, accountID, &openingCategory.ID, account.Currency, openedDay, amount)
	if _, err := s.txnRepo.InsertTransaction(ctx, tx, &openingTxn); err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to post the opening transaction: %w", err)
	}

	account.ID = accountID
	if err := s.UpdateAccountCashBalance(ctx, tx, account, openedDay); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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
	exAcc, _, err := s.repo.FindAccountByIDWithOpening(ctx, tx, id, userID)
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

	// Resolve new relations from req
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

		// Delete all existing snapshots for this account
		err = s.balanceRepo.DeleteAccountSnapshots(ctx, tx, id)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to delete existing snapshots: %w", err)
		}

		// The opening transaction carries the starting amount, so retiming it is the
		// whole move.
		if err := s.txnRepo.RetimeOpeningTransactions(ctx, tx, id, newOpenedAt); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to move the opening transaction: %w", err)
		}

		// Rebuild from the earlier of the two days, so the day it left is recomputed too.
		rebuildFrom := newOpenedAt
		if oldOpenedDay := exAcc.OpenedAt.UTC().Truncate(24 * time.Hour); oldOpenedDay.Before(rebuildFrom) {
			rebuildFrom = oldOpenedDay
		}

		if err := s.balanceRepo.RebuildBalances(ctx, tx, userID, id, exAcc.Currency, rebuildFrom); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to rebuild balances from the new opened date: %w", err)
		}
	}

	if req.CreditLimit != nil && req.CreditLimit.IsZero() {
		req.CreditLimit = nil
	}

	if (exAcc.CreditLimit != nil && req.CreditLimit == nil) || req.CreditLimit != nil {
		latestBal, err := s.balanceRepo.FindLatestBalance(ctx, tx, exAcc.ID, userID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		if req.CreditLimit == nil && !latestBal.IsPositive() {
			tx.Rollback()
			return 0, errors.New("cannot remove credit limit while account balance is not positive")
		} else if req.CreditLimit != nil && latestBal.IsNegative() && req.CreditLimit.LessThanOrEqual(latestBal.Neg()) {
			tx.Rollback()
			return 0, errors.New("credit limit must exceed current negative balance")
		}
	}

	acc := &models.Account{
		ID:            id,
		Name:          req.Name,
		Currency:      exAcc.Currency,
		AccountTypeID: newAccType.ID,
		IsActive:      exAcc.IsActive,
		UserID:        userID,
		OpenedAt:      newOpenedAt,
	}
	if newAccType.Classification != "liability" {
		acc.CreditLimit = req.CreditLimit
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

		latestBalance, err := s.balanceRepo.FindLatestBalance(ctx, tx, exAcc.ID, userID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		// Match sign conventions
		isLiability := strings.EqualFold(newAccType.Classification, "liability")

		if !isLiability {
			floor := decimal.Zero
			if req.CreditLimit != nil {
				floor = req.CreditLimit.Neg()
			}
			if desired.LessThan(floor) {
				tx.Rollback()
				return 0, utils.AccountLimitError(desired, &models.Account{
					AccountType: models.AccountType{Classification: "asset"},
					CreditLimit: req.CreditLimit,
				})
			}
		}

		delta = desired.Sub(latestBalance)

		if delta.IsNegative() {
			uncategorized, err := s.savingsRepo.GetUncategorizedBalance(ctx, tx, exAcc.ID, userID)
			if err != nil {
				tx.Rollback()
				return 0, err
			}
			if err := utils.CheckGoalAllocation(delta.Neg(), uncategorized, newAccType.Classification); err != nil {
				tx.Rollback()
				return 0, err
			}
		}
		signed := delta

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
				Direction:       txnType,
				Amount:          amount,
				Currency:        exAcc.Currency,
				TxnDate:         time.Now().UTC(),
				Description:     &desc,
				TransactionType: models.TxnTypeAdjustment,
			}

			if _, err := s.txnRepo.InsertTransaction(ctx, tx, txn); err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("failed to post adjustment transaction: %w", err)
			}

			err = s.UpdateAccountCashBalance(ctx, tx, acc, txn.TxnDate)
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
			s.logger.Error("balance change logging failed",
				zap.Error(err), zap.Int64("account_id", exAcc.ID))
		}
	}

	if changes.HasChanges() {
		changes.Stamp("id", strconv.FormatInt(acc.ID, 10))
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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
	exAcc, err := s.repo.FindAccountByID(ctx, tx, id, userID, false, true)
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

	if changes.HasChanges() {
		event := "deactivate"
		if acc.IsActive {
			event = "restore"
		}

		changes.Stamp("id", strconv.FormatInt(acc.ID, 10))
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       event,
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
	acc, err := s.repo.FindAccountByID(ctx, tx, id, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	if !acc.Balance.TotalBalance.IsZero() {
		tx.Rollback()
		return ErrAccountNotEmpty
	}

	// Close it
	if err := s.repo.CloseAccount(ctx, tx, acc.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Materialize a real snapshot for today so charts don’t copy yesterday’s value
	// Upsert for the just-closed account
	_ = s.balanceRepo.RebuildDailyRange(ctx, tx, userID, acc.ID, acc.Currency, today, today)

	// Upsert for all still-open accounts for today (so the view has a “today” row)
	openAccs, err := s.repo.FindAllAccounts(ctx, tx, userID, false, false)
	if err == nil {
		for _, a := range openAccs {
			_ = s.balanceRepo.RebuildDailyRange(ctx, tx, userID, a.ID, a.Currency, today, today)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	utils.CompareChanges("", strconv.FormatInt(acc.ID, 10), changes, "id")
	utils.CompareChanges(acc.Name, "", changes, "account")
	utils.CompareChanges(acc.AccountType.Type, "", changes, "type")
	utils.CompareChanges(acc.AccountType.Subtype, "", changes, "sub_type")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *AccountService) FetchAccountsForUser(ctx context.Context, userID int64) ([]models.AccountLookup, error) {
	return s.repo.FindAccountsForUser(ctx, nil, userID)
}

// A hard delete with no restore path. Closing an account is the normal route;
// this is the override.
func (s *AccountService) PurgeAccount(ctx context.Context, actorID, accountID int64) error {

	acc, err := s.repo.FindAccountForPurge(ctx, nil, accountID)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
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

	if err := s.repo.PurgeAccount(ctx, tx, acc.ID, acc.UserID); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.rebuildUserHistory(ctx, tx, acc.UserID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	utils.CompareChanges("", strconv.FormatInt(acc.ID, 10), changes, "id")
	utils.CompareChanges(acc.Name, "", changes, "account")
	utils.CompareChanges(strconv.FormatInt(acc.UserID, 10), "", changes, "owner")

	if changes.IsEmpty() {
		return nil
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "purge",
		Category:    "account",
		Description: nil,
		Payload:     changes,
		Causer:      &actorID,
	})
}

// A purge takes both legs of its transfers with it, so any other account of the
// user can be left short. Re-chain and re-materialize every one of them from its
// opening day.
func (s *AccountService) rebuildUserHistory(ctx context.Context, tx *gorm.DB, userID int64) error {

	accounts, err := s.repo.FindAllAccountsForRebuild(ctx, tx, userID)
	if err != nil {
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	earliest := today

	for _, acc := range accounts {
		from, err := s.repo.GetAccountOpeningAsOf(ctx, tx, acc.ID)
		if err != nil {
			// No balance rows means no history to rebuild.
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}

		if from.Before(earliest) {
			earliest = from
		}

		if err := s.balanceRepo.RebuildBalances(ctx, tx, userID, acc.ID, acc.Currency, from); err != nil {
			return err
		}
	}

	return s.balanceRepo.UpdateSnapshotMarketValues(ctx, tx, userID, utils.SnapshotRecomputeFrom(earliest))
}

func (s *AccountService) UpdateAccountCashBalance(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time) error {
	return s.balanceRepo.RebuildBalances(ctx, tx, acc.UserID, acc.ID, acc.Currency, asOf)
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

	accounts, err := s.repo.FindAllAccounts(ctx, tx, userID, true, false)
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
	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	loc := time.UTC
	if err == nil && settings != nil {
		if l, e := time.LoadLocation(settings.Timezone); e == nil && l != nil {
			loc = l
		}
	}
	now := time.Now().In(loc)
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, loc)

	var dfrom time.Time
	var dto time.Time
	var parseErr error

	if strings.TrimSpace(to) == "" {
		dto = today
	} else {
		dto, parseErr = time.Parse("2006-01-02", to)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date: %w", parseErr)
		}
	}

	if strings.TrimSpace(from) != "" {
		dfrom, parseErr = time.Parse("2006-01-02", from)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date: %w", parseErr)
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
	return s.balanceRepo.RebuildDailyRange(
		ctx,
		tx,
		acc.UserID,
		acc.ID,
		acc.Currency,
		dfrom,
		dto,
	)
}

func (s *AccountService) UpdateDailyCashNoSnapshot(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, txnType string, amt decimal.Decimal) error {
	amt = amt.Round(4)
	if strings.ToLower(txnType) == "expense" {
		amt = amt.Neg()
	}
	return s.balanceRepo.ApplyDelta(ctx, tx, acc.ID, amt)
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

	utils.CompareChanges("", strconv.FormatInt(acc.ID, 10), changes, "id")
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
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

	utils.CompareChanges("", strconv.FormatInt(acc.ID, 10), changes, "id")
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
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *AccountService) FetchAccountsWithDefaults(ctx context.Context, userID int64) ([]models.Account, error) {
	return s.repo.FindAccountsWithDefaults(ctx, nil, userID)
}

func (s *AccountService) FetchAccountTypesWithoutDefaults(ctx context.Context, userID int64) ([]models.AccountType, error) {
	return s.repo.FindAccountTypesWithoutDefaults(ctx, nil, userID)
}

func (s *AccountService) SetDefaultAccount(ctx context.Context, userID, accountID int64) error {
	return s.updateDefaultAccount(ctx, userID, accountID, true)
}

func (s *AccountService) UnsetDefaultAccount(ctx context.Context, userID, accountID int64) error {
	return s.updateDefaultAccount(ctx, userID, accountID, false)
}

func (s *AccountService) updateDefaultAccount(ctx context.Context, userID, accountID int64, setAsDefault bool) error {
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

	// Confirm account exists
	account, err := s.repo.FindAccountByID(ctx, tx, accountID, userID, false)
	if err != nil {
		return err
	}

	// Only check for existing default when setting (not unsetting)
	if setAsDefault {
		hasDefault, err := s.repo.HasDefaultForAccountType(ctx, tx, userID, account.AccountTypeID)
		if err != nil {
			return err
		}

		if hasDefault {
			return fmt.Errorf("a default account already exists for this account type")
		}
	}

	changes := utils.InitChanges()
	utils.CompareChanges(strconv.FormatBool(account.IsDefault), strconv.FormatBool(setAsDefault), changes, "default")

	err = s.repo.UpdateDefaultAccount(ctx, tx, *account, setAsDefault)
	if err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if changes.HasChanges() {
		changes.Stamp("id", strconv.FormatInt(account.ID, 10))
		if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       "update",
			Category:    "account",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *AccountService) UpdateSnapshotMarketValues(ctx context.Context, userID int64, from time.Time) error {
	return s.balanceRepo.UpdateSnapshotMarketValues(ctx, nil, userID, utils.SnapshotRecomputeFrom(from))
}

func (s *AccountService) SyncForUser(ctx context.Context, userID int64) error {
	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	today := now.Truncate(24 * time.Hour)

	exists, err := s.balanceRepo.HasSnapshotForDate(ctx, userID, today)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	yesterday := today.AddDate(0, 0, -1)
	if err := s.BackfillBalancesForUser(ctx, userID, yesterday.Format("2006-01-02"), today.Format("2006-01-02")); err != nil {
		return err
	}

	return s.balanceRepo.UpdateSnapshotMarketValues(ctx, nil, userID, &today)
}

func (s *AccountService) RecalculateAssetPnL(ctx context.Context, userID, assetID int64) error {
	if s.priceFetchClient != nil {
		asset, err := s.investmentRepo.FindInvestmentAssetByID(ctx, nil, assetID, userID)
		if err != nil {
			return err
		}
		priceData, err := s.priceFetchClient.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err == nil && priceData != nil && priceData.Price > 0 {
			today := time.Now().UTC().Truncate(24 * time.Hour)
			price := decimal.NewFromFloat(priceData.Price)
			if err := s.investmentRepo.UpsertTickerPrice(ctx, nil, []models.TickerPriceHistory{
				{Ticker: asset.Ticker, AsOf: today, Price: price, Currency: priceData.Currency},
			}); err != nil {
				return err
			}
		}
	}
	return s.investmentRepo.RecalculateAssetFromTrades(ctx, nil, assetID, userID)
}

func (s *AccountService) GetAssetIDsForAccount(ctx context.Context, userID, accountID int64) ([]int64, error) {
	return s.investmentRepo.GetAssetIDsForAccount(ctx, nil, accountID, userID)
}

func (s *AccountService) SyncAssetPnL(ctx context.Context, userID, assetID int64) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.RecalculateAssetPnLArgs{UserID: userID, AssetID: &assetID})
}

func (s *AccountService) SyncAccountPnL(ctx context.Context, userID, accountID int64) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.RecalculateAssetPnLArgs{UserID: userID, AccountID: &accountID})
}

func (s *AccountService) QueueAccountMerge(ctx context.Context, userID, sourceID, destinationID int64) error {
	srcAcc, dstAcc, err := s.resolveAccountMerge(ctx, nil, userID, sourceID, destinationID)
	if err != nil {
		return err
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.MergeAccountsArgs{
		UserID:                       userID,
		InternalSourceAccountID:      srcAcc.ID,
		InternalDestinationAccountID: dstAcc.ID,
		SourceAccount:                srcAcc.Name,
		DestinationAccount:           dstAcc.Name,
	})
}

func (s *AccountService) resolveAccountMerge(ctx context.Context, tx *gorm.DB, userID, sourceID, destinationID int64) (*models.Account, *models.Account, error) {
	if sourceID == destinationID {
		return nil, nil, errors.New("source and destination accounts must be different")
	}

	srcAcc, err := s.repo.FindAccountByID(ctx, tx, sourceID, userID, false, true)
	if err != nil {
		return nil, nil, fmt.Errorf("source account not found: %w", err)
	}
	dstAcc, err := s.repo.FindAccountByID(ctx, tx, destinationID, userID, false, true)
	if err != nil {
		return nil, nil, fmt.Errorf("destination account not found: %w", err)
	}

	// Guard: investment and crypto accounts require the same type and sub-type (no cross-merging)
	srcType := strings.ToLower(srcAcc.AccountType.Type)
	if srcType == "investment" || srcType == "crypto" {
		if srcAcc.AccountType.Type != dstAcc.AccountType.Type || srcAcc.AccountType.Subtype != dstAcc.AccountType.Subtype {
			return nil, nil, fmt.Errorf(
				"investment/crypto accounts can only be merged into an account with the same type and sub-type (%s / %s)",
				srcAcc.AccountType.Type, srcAcc.AccountType.Subtype,
			)
		}
	}

	// Guard: liability accounts can only be merged into other liability accounts
	srcIsLiability := strings.ToLower(srcAcc.AccountType.Classification) == "liability"
	dstIsLiability := strings.ToLower(dstAcc.AccountType.Classification) == "liability"
	if srcIsLiability != dstIsLiability {
		return nil, nil, fmt.Errorf("liability accounts can only be merged into other liability accounts")
	}

	return srcAcc, dstAcc, nil
}

func (s *AccountService) MergeAccount(ctx context.Context, userID, sourceID, destinationID int64) error {
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

	srcAcc, dstAcc, err := s.resolveAccountMerge(ctx, tx, userID, sourceID, destinationID)
	if err != nil {
		tx.Rollback()
		return err
	}

	srcType := strings.ToLower(srcAcc.AccountType.Type)

	// Count transactions being moved before the bulk update
	txnCount, err := s.txnRepo.CountTransactions(ctx, tx, userID, []utils.Filter{}, false, &sourceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Resolve the earliest source transaction date for the balance rebuild
	earliestTxnDate, err := s.repo.FindEarliestTransactionDate(ctx, tx, sourceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Soft-delete transfers between the two accounts and flag their transactions as adjustments
	transfers, err := s.txnRepo.FindTransfersBetweenAccounts(ctx, tx, sourceID, destinationID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	now := time.Now().UTC()
	for _, t := range transfers {
		txnIDs := []int64{t.TransactionInflowID, t.TransactionOutflowID}
		if err := tx.Model(&models.Transaction{}).
			Where("id IN ?", txnIDs).
			Updates(map[string]any{
				"transaction_type": models.TxnTypeAdjustment,
				"updated_at":       now,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Model(&models.Transfer{}).Where("id = ?", t.ID).Updates(map[string]any{
			"deleted_at": now,
			"updated_at": now,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Reassign transactions from source to destination
	if err := s.txnRepo.BulkUpdateTransactionAccountID(ctx, tx, sourceID, destinationID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Reassign templates from source to destination
	if err := s.txnRepo.BulkUpdateTemplateAccountIDs(ctx, tx, sourceID, destinationID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Reassign investment assets from source to destination (investment and crypto only)
	if srcType == "investment" || srcType == "crypto" {
		if err := s.investmentRepo.BulkUpdateAssetAccountID(ctx, tx, sourceID, destinationID, userID); err != nil {
			tx.Rollback()
			return err
		}
	}

	dstOpeningDay := dstAcc.OpenedAt.UTC().Truncate(24 * time.Hour)

	// If source transactions predate the destination's opening date, push opened_at back
	// to one day before the earliest transaction so the validator is satisfied.
	if earliestTxnDate != nil && earliestTxnDate.Before(dstOpeningDay) {
		dstOpeningDay = earliestTxnDate.AddDate(0, 0, -1)
		if err := tx.WithContext(ctx).Model(&models.Account{}).
			Where("id = ? AND user_id = ?", destinationID, userID).
			Update("opened_at", dstOpeningDay).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// The transactions moved, so both balances have to be recounted. The source is
	// left with none. These stay split from the daily rebuild below because
	// CloseAccount runs between them, so the pair cannot be one call.
	srcOpeningDate := srcAcc.OpenedAt.UTC().Truncate(24 * time.Hour)
	if err := s.balanceRepo.RecomputeFromTransactions(ctx, tx, sourceID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.balanceRepo.RecomputeFromTransactions(ctx, tx, destinationID); err != nil {
		tx.Rollback()
		return err
	}

	// Close the source account
	if err := s.repo.CloseAccount(ctx, tx, sourceID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Rebuild snapshots for source and dest inside the same transaction.
	for _, pair := range []struct {
		id       int64
		currency string
		from     time.Time
	}{
		{sourceID, srcAcc.Currency, srcOpeningDate},
		{destinationID, dstAcc.Currency, dstOpeningDay},
	} {
		if err := s.balanceRepo.RebuildDailyRange(ctx, tx, userID, pair.id, pair.currency, pair.from, now.Truncate(24*time.Hour)); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(srcAcc.Name, "", changes, "source_account")
	utils.CompareChanges("", dstAcc.Name, changes, "destination_account")
	utils.CompareChanges("", strconv.FormatInt(txnCount, 10), changes, "transactions_moved")
	utils.CompareChanges("", strconv.FormatInt(int64(len(transfers)), 10), changes, "transfers_removed")
	changes.Stamp("id", strconv.FormatInt(dstAcc.ID, 10))

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "merge",
		Category:    "account",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		s.logger.Error("account merge activity log failed",
			zap.Error(err), zap.Int64("source_id", sourceID), zap.Int64("destination_id", destinationID))
	}

	return nil
}
