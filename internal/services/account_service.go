package services

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type AccountService struct {
	Config  *config.Config
	Ctx     *DefaultServiceContext
	Repo    *repositories.AccountRepository
	TxnRepo *repositories.TransactionRepository
}

func NewAccountService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
) *AccountService {
	return &AccountService{
		Ctx:     ctx,
		Config:  cfg,
		Repo:    repo,
		TxnRepo: txnRepo,
	}
}

func (s *AccountService) LogBalanceChange(account *models.Account, userID int64, change decimal.Decimal) error {
	newBalance, err := s.Repo.FindBalanceForAccountID(nil, account.ID)
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

	return s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "update",
		Category:    "balance",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
}

func (s *AccountService) FetchAccountsPaginated(userID int64, p utils.PaginationParams, includeInactive bool) ([]models.Account, *utils.Paginator, error) {

	totalRecords, err := s.Repo.CountAccounts(userID, p.Filters, includeInactive)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage
	records, err := s.Repo.FindAccounts(userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeInactive)
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

func (s *AccountService) FetchAccountByID(userID int64, id int64) (*models.Account, error) {

	record, err := s.Repo.FindAccountByID(nil, id, userID, true)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *AccountService) FetchAllAccounts(userID int64, includeInactive bool) ([]models.Account, error) {
	return s.Repo.FindAllAccounts(nil, userID, includeInactive)
}

func (s *AccountService) FetchAllAccountTypes() ([]models.AccountType, error) {
	return s.Repo.FindAllAccountTypes(nil, nil)
}

func (s *AccountService) InsertAccount(userID int64, req *models.AccountReq) error {

	changes := utils.InitChanges()

	if req.Classification == "asset" && req.Balance.LessThan(decimal.NewFromInt(0)) {
		return errors.New("provided initial balance cannot be negative")
	}

	accCount, err := s.Repo.CountAccounts(userID, nil, false)
	if err != nil {
		return err
	}

	maxAcc, err := s.Ctx.SettingsRepo.FetchMaxAccountsForUser(nil)
	if err != nil {
		return err
	}

	if accCount >= maxAcc {
		return fmt.Errorf("you can only have %d active accounts", maxAcc)
	}

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

	accType, err := s.Repo.FindAccountTypeByID(tx, req.AccountTypeID)
	if err != nil {
		return fmt.Errorf("can't find account_type for given id %w", err)
	}

	account := &models.Account{
		Name:          req.Name,
		Currency:      models.DefaultCurrency,
		AccountTypeID: accType.ID,
		UserID:        userID,
	}

	balanceAmountString := req.Balance.StringFixed(2)

	utils.CompareChanges("", account.Name, changes, "name")
	utils.CompareChanges("", accType.Type, changes, "account_type")
	utils.CompareChanges("", accType.Subtype, changes, "account_subtype")
	utils.CompareChanges("", account.Currency, changes, "currency")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")

	accountID, err := s.Repo.InsertAccount(tx, account)
	if err != nil {
		tx.Rollback()
		return err
	}

	amount := req.Balance.Round(4)

	if accType.Classification == "liability" {
		amount = amount.Neg()
	}

	balance := &models.Balance{
		AccountID:    accountID,
		Currency:     models.DefaultCurrency,
		StartBalance: amount,
		AsOf:         time.Now(),
	}

	_, err = s.Repo.InsertBalance(tx, balance)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "account",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *AccountService) UpdateAccount(userID int64, id int64, req *models.AccountReq) error {

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

	// Load record
	exAcc, err := s.Repo.FindAccountByID(tx, id, userID, true)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	if !exAcc.IsActive {
		return errors.New("can't update non-active account")
	}

	// Load existing relations for comparison
	exAccType, err := s.Repo.FindAccountTypeByID(tx, exAcc.AccountTypeID)
	if err != nil {
		return fmt.Errorf("can't find account type with given id %w", err)
	}

	// Resolve new relations  from req
	newAccType, err := s.Repo.FindAccountTypeByID(tx, req.AccountTypeID)
	if err != nil {
		return fmt.Errorf("can't find account type with given id %w", err)
	}

	acc := &models.Account{
		ID:            id,
		Name:          req.Name,
		Currency:      models.DefaultCurrency,
		AccountTypeID: newAccType.ID,
		IsActive:      exAcc.IsActive,
		UserID:        userID,
	}

	changes := utils.InitChanges()

	utils.CompareChanges(exAcc.Name, acc.Name, changes, "name")
	utils.CompareChanges(exAccType.Type, newAccType.Type, changes, "account_type")
	utils.CompareChanges(exAccType.Subtype, newAccType.Subtype, changes, "account_subtype")
	utils.CompareChanges(exAcc.Currency, acc.Currency, changes, "currency")

	var delta decimal.Decimal

	if req.Balance != nil {

		desired, err := decimal.NewFromString(req.Balance.String())
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("invalid balance value: %w", err)
		}

		// Current end balance from snapshot
		asOf := time.Now().UTC() // or your canonical TZ
		current, err := utils.GetEndBalanceAsOf(tx, exAcc.ID, asOf)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Match sign conventions
		isLiability := strings.EqualFold(newAccType.Type, "liability")

		delta = desired.Sub(current)
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

			category, err := s.TxnRepo.FindCategoryByClassification(tx, "adjustment", &userID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("can't find adjustment category: %w", err)
			}

			txn := &models.Transaction{
				UserID:          userID,
				AccountID:       exAcc.ID,
				CategoryID:      &category.ID,
				TransactionType: txnType,
				Amount:          amount,
				Currency:        exAcc.Currency,
				TxnDate:         time.Now(),
				Description:     &desc,
				IsAdjustment:    true,
			}

			if _, err := s.TxnRepo.InsertTransaction(tx, txn); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to post adjustment transaction: %w", err)
			}

			err = s.UpdateAccountCashBalance(tx, acc, txn.TxnDate, txnType, amount)
			if err != nil {
				tx.Rollback()
				return err
			}

		}
	}

	_, err = s.Repo.UpdateAccount(tx, acc)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// balance log (with the new end_balance)
	if req.Balance != nil {
		accForLog := &models.Account{ID: exAcc.ID, Name: acc.Name, Currency: exAcc.Currency}
		if err := s.LogBalanceChange(accForLog, userID, delta); err != nil {
			s.Ctx.Logger.Warn("Balance change logging failed")
		}
	}

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
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

func (s *AccountService) UpdateAccountCashBalance(
	tx *gorm.DB,
	acc *models.Account,
	asOf time.Time,
	transactionType string,
	amount decimal.Decimal,
) error {
	// ensure daily balance row exists for asOf
	if err := s.Repo.EnsureDailyBalanceRow(tx, acc.ID, asOf, acc.Currency); err != nil {
		return err
	}

	amount = amount.Round(4)

	// increment the correct field on balances(as_of)
	switch strings.ToLower(transactionType) {
	case "expense":
		// expense decreases cash => goes to cash_outflows
		if err := s.Repo.AddToDailyBalance(tx, acc.ID, asOf, "cash_outflows", amount); err != nil {
			return err
		}
	default:
		// income increases cash => goes to cash_inflows
		if err := s.Repo.AddToDailyBalance(tx, acc.ID, asOf, "cash_inflows", amount); err != nil {
			return err
		}
	}

	return nil
}

func (s *AccountService) ToggleAccountActiveState(userID int64, id int64) error {

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
	exAcc, err := s.Repo.FindAccountByID(tx, id, userID, false)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	accCount, err := s.Repo.CountAccounts(userID, nil, false)
	if err != nil {
		return err
	}

	maxAcc, err := s.Ctx.SettingsRepo.FetchMaxAccountsForUser(nil)
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

	_, err = s.Repo.UpdateAccount(tx, acc)
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

func (s *AccountService) CloseAccount(userID int64, id int64) error {

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

	// Load the account
	acc, err := s.Repo.FindAccountByID(tx, id, userID, false)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	// Close it
	if err := s.Repo.CloseAccount(tx, acc.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	utils.CompareChanges(acc.Name, "", changes, "account")
	utils.CompareChanges(acc.AccountType.Type, "", changes, "type")
	utils.CompareChanges(acc.AccountType.Subtype, "", changes, "sub_type")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "delete",
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

func (s *AccountService) resolveUserDateRange(tx *gorm.DB, userID int64, from, to string) (time.Time, time.Time, error) {
	today := time.Now().Truncate(24 * time.Hour)

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
		fb, err := s.Repo.GetUserFirstBalanceDate(tx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		ft, err := s.Repo.GetUserFirstTxnDate(tx, userID)
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

func (s *AccountService) backfillAccountRange(
	tx *gorm.DB,
	acc *models.Account,
	dfrom, dto time.Time,
) error {
	return s.Repo.UpsertSnapshotsFromBalances(
		tx,
		acc.UserID,
		acc.ID,
		acc.Currency,
		dfrom,
		dto,
	)
}

func (s *AccountService) BackfillBalancesForUser(userID int64, from, to string) error {
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

	accounts, err := s.Repo.FindAllAccounts(tx, userID, true) // unchanged
	if err != nil {
		tx.Rollback()
		return err
	}
	if len(accounts) == 0 {
		return tx.Commit().Error
	}

	dfrom, dto, err := s.resolveUserDateRange(tx, userID, from, to) // unchanged
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, acc := range accounts {
		if err := s.backfillAccountRange(tx, &acc, dfrom, dto); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
