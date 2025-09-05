package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
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

func (s *AccountService) LogBalanceChange(account *models.Account, user *models.User, change decimal.Decimal) error {
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
		Causer:      user,
	})
}

func (s *AccountService) FetchAccountsPaginated(c *gin.Context) ([]models.Account, *utils.Paginator, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, nil, err
	}

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)
	yearParam := queryParams.Get("year")

	// Get the current year
	currentYear := time.Now().Year()

	// Convert yearParam to integer
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 { // Ensure year is valid
		year = currentYear // Default to current year if invalid
	}

	totalRecords, err := s.Repo.CountAccounts(user, year, paginationParams.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	records, err := s.Repo.FindAccounts(user, year, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, paginationParams.Filters)
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
		CurrentPage:  paginationParams.PageNumber,
		RowsPerPage:  paginationParams.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *AccountService) FetchAccountByID(c *gin.Context, id int64) (*models.Account, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}

	record, err := s.Repo.FindAccountByID(nil, id, user.ID, true)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *AccountService) FetchAllAccounts(c *gin.Context) ([]models.Account, error) {
	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.Repo.FindAllAccounts(user)
}

func (s *AccountService) FetchAllAccountTypes(c *gin.Context) ([]models.AccountType, error) {
	return s.Repo.FindAllAccountTypes(nil)
}

func (s *AccountService) InsertAccount(c *gin.Context, req *models.AccountReq) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	if req.Balance.LessThan(decimal.NewFromInt(0)) {
		return errors.New("provided initial balance cannot be negative")
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
		UserID:        user.ID,
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
		Causer:      user,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *AccountService) UpdateAccount(c *gin.Context, id int64, req *models.AccountReq) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
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

	// Load record
	exAcc, err := s.Repo.FindAccountByID(tx, id, user.ID, true)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
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
		UserID:        user.ID,
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

			category, err := s.TxnRepo.FindCategoryByClassification(tx, "adjustment", &user.ID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("can't find adjustment category: %w", err)
			}

			txn := &models.Transaction{
				UserID:          user.ID,
				AccountID:       exAcc.ID,
				CategoryID:      &category.ID,
				TransactionType: txnType,
				Amount:          amount,
				Currency:        exAcc.Currency,
				TxnDate:         time.Now(),
				Description:     &desc,
			}

			if _, err := s.TxnRepo.InsertTransaction(tx, txn); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to post adjustment transaction: %w", err)
			}

			err = s.UpdateAccountCashBalance(tx, acc, txnType, amount)
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
		if err := s.LogBalanceChange(accForLog, user, delta); err != nil {
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
			Causer:      user,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *AccountService) UpdateAccountCashBalance(tx *gorm.DB, acc *models.Account, transactionType string, amount decimal.Decimal) error {

	accBalance, err := s.Repo.FindBalanceForAccountID(tx, acc.ID)
	if err != nil {
		return fmt.Errorf("can't find balance for given account id %w", err)
	}

	amount = amount.Round(4)

	switch transactionType {
	case "expense":
		accBalance.CashOutflows = accBalance.CashOutflows.Add(amount)
	default:
		accBalance.CashInflows = accBalance.CashInflows.Add(amount)
	}

	_, err = s.Repo.UpdateBalance(tx, accBalance)
	if err != nil {
		return err
	}

	return nil
}
