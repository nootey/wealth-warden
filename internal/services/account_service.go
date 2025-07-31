package services

import (
	"errors"
	"github.com/gin-gonic/gin"
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

func (s *AccountService) FetchAllAccountTypes(c *gin.Context) ([]models.AccountType, error) {
	return s.Repo.FindAllAccountTypes(nil)
}

func (s *AccountService) InsertAccount(c *gin.Context, newRecord *models.CreateAccountRequest) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	if newRecord.Balance < 0 {
		return errors.New("initial balance cannot be negative")
	}

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Always rollback unless explicitly committed
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if tx.Error == nil {
			tx.Rollback()
		}
	}()

	account := &models.Account{
		Name:          newRecord.Name,
		Currency:      models.DefaultCurrency,
		AccountTypeID: newRecord.AccountTypeID,
		UserID:        user.ID,
	}

	balanceAmountString := strconv.FormatFloat(newRecord.Balance, 'f', 2, 64)

	utils.CompareChanges("", account.Name, changes, "name")
	utils.CompareChanges("", newRecord.Type, changes, "account_type")
	utils.CompareChanges("", newRecord.Subtype, changes, "account_subtype")
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
		StartBalance: newRecord.Balance,
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
