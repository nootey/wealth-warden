package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type AccountService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.AccountRepository
}

func NewAccountService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.AccountRepository,
) *AccountService {
	return &AccountService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
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

func (s *AccountService) InsertAccount(c *gin.Context, req *models.AccountCreateReq) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	if req.Balance < 0 {
		return errors.New("initial balance cannot be negative")
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

	balanceAmountString := strconv.FormatFloat(req.Balance, 'f', 2, 64)

	utils.CompareChanges("", account.Name, changes, "name")
	utils.CompareChanges("", accType.Type, changes, "account_type")
	utils.CompareChanges("", utils.SafeString(accType.Subtype), changes, "account_subtype")
	utils.CompareChanges("", account.Currency, changes, "currency")
	utils.CompareChanges("", balanceAmountString, changes, "current_balance")

	accountID, err := s.Repo.InsertAccount(tx, account)
	if err != nil {
		tx.Rollback()
		return err
	}

	balance := &models.Balance{
		AccountID:    accountID,
		Currency:     models.DefaultCurrency,
		StartBalance: req.Balance,
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
		LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
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

func (s *AccountService) UpdateAccountCashBalance(tx *gorm.DB, acc *models.Account, transactionType string, amount float64) error {

	accBalance, err := s.Repo.FindBalanceForAccountID(tx, acc.ID)
	if err != nil {
		return fmt.Errorf("can't find balance for given account id %w", err)
	}

	switch transactionType {
	case "expense":
		accBalance.CashOutflows += amount
	default:
		accBalance.CashInflows += amount
	}

	_, err = s.Repo.UpdateBalance(tx, accBalance)
	if err != nil {
		return err
	}

	return nil
}
