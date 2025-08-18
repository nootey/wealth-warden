package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
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

func (s *TransactionService) FetchTransactionsPaginated(c *gin.Context) ([]models.Transaction, *utils.Paginator, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, nil, err
	}

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)

	totalRecords, err := s.Repo.CountTransactions(user, paginationParams.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	records, err := s.Repo.FindTransactions(user, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, paginationParams.Filters)
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

func (s *TransactionService) FetchTransactionByID(c *gin.Context, id int64) (*models.Transaction, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}

	record, err := s.Repo.FindTransactionByID(nil, id, user.ID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TransactionService) FetchAllCategories(c *gin.Context) ([]models.Category, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}

	categories, err := s.Repo.FindAllCategories(user)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *TransactionService) InsertTransaction(c *gin.Context, req *models.TransactionReq) error {

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

	account, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, user.ID)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	var category models.Category
	if req.CategoryID != nil {
		category, err = s.Repo.FindCategoryByID(tx, *req.CategoryID, &user.ID)
		if err != nil {
			return fmt.Errorf("can't find category with given id %w", err)
		}
	} else {
		category, err = s.Repo.FindCategoryByClassification(tx, "uncategorized", &user.ID)
		if err != nil {
			return fmt.Errorf("can't find default category %w", err)
		}
	}

	tr := models.Transaction{
		UserID:          user.ID,
		AccountID:       account.ID,
		CategoryID:      &category.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         req.TxnDate,
		Description:     req.Description,
	}

	_, err = s.Repo.InsertTransaction(tx, tr)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.AccountService.UpdateAccountCashBalance(tx, &account, tr.TransactionType, tr.Amount)
	if err != nil {
		tx.Rollback()
		return err
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
		LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      user,
	})
	if err != nil {
		return err
	}

	// Dispatch balance activity log
	changes2 := utils.InitChanges()

	// Re-fetch account balance to get the computed end_balance
	newBalance, err := s.AccountService.Repo.FindBalanceForAccountID(nil, account.ID)
	if err != nil {
		return err
	}

	endBalanceString := newBalance.EndBalance.StringFixed(2)

	var change decimal.Decimal
	switch tr.TransactionType {
	case "expense":
		change = tr.Amount.Neg()
	default:
		change = tr.Amount
	}

	startBalance := newBalance.EndBalance.Sub(change)

	changeAmountString := change.StringFixed(2)
	startBalanceString := startBalance.StringFixed(2)

	utils.CompareChanges("", account.Name, changes2, "account")
	utils.CompareChanges("", changeAmountString, changes2, "change")
	utils.CompareChanges("", startBalanceString, changes2, "start_balance")
	utils.CompareChanges("", endBalanceString, changes2, "end_balance")
	utils.CompareChanges("", account.Currency, changes2, "currency")

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
		Logger:      s.Ctx.Logger,
		Event:       "update",
		Category:    "balance",
		Description: nil,
		Payload:     changes2,
		Causer:      user,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) UpdateTransaction(c *gin.Context, id int64, req *models.TransactionReq) error {

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

	exTr, err := s.Repo.FindTransactionByID(tx, id, user.ID)
	if err != nil {
		return fmt.Errorf("can't find transaction with given id %w", err)
	}

	// Load existing relations for comparison
	oldAccount, err := s.AccountService.Repo.FindAccountByID(tx, exTr.AccountID, user.ID)
	if err != nil {
		return fmt.Errorf("can't find existing account: %w", err)
	}
	var oldCategory models.Category
	if exTr.CategoryID != nil {
		oldCategory, _ = s.Repo.FindCategoryByID(tx, *exTr.CategoryID, &user.ID)
	}

	// Resolve new relations  from req
	newAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, user.ID)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	var newCategory models.Category
	if req.CategoryID != nil {
		newCategory, err = s.Repo.FindCategoryByID(tx, *req.CategoryID, &user.ID)
		if err != nil {
			return fmt.Errorf("can't find category with given id %w", err)
		}
	} else {
		newCategory, err = s.Repo.FindCategoryByClassification(tx, "uncategorized", &user.ID)
		if err != nil {
			return fmt.Errorf("can't find default category %w", err)
		}
	}

	tr := models.Transaction{
		ID:              exTr.ID,
		UserID:          user.ID,
		AccountID:       newAccount.ID,
		CategoryID:      &newCategory.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Currency:        exTr.Currency,
		TxnDate:         req.TxnDate,
		Description:     req.Description,
	}

	_, err = s.Repo.UpdateTransaction(tx, tr)
	if err != nil {
		tx.Rollback()
		return err
	}

	signed := func(tt string, amt decimal.Decimal) decimal.Decimal {
		switch strings.ToLower(tt) {
		case "expense":
			return amt.Neg()
		default:
			return amt
		}
	}
	oldEffect := signed(exTr.TransactionType, exTr.Amount)
	newEffect := signed(tr.TransactionType, tr.Amount)

	if oldAccount.ID != newAccount.ID {
		if !oldEffect.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(tx, &oldAccount,
				map[bool]string{true: "income", false: "expense"}[oldEffect.IsNegative()],
				oldEffect.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
		if !newEffect.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(tx, &newAccount,
				map[bool]string{true: "expense", false: "income"}[newEffect.IsNegative()],
				newEffect.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	} else {
		delta := newEffect.Sub(oldEffect)
		if !delta.IsZero() {
			if err := s.AccountService.UpdateAccountCashBalance(tx, &newAccount,
				map[bool]string{true: "expense", false: "income"}[delta.IsNegative()],
				delta.Abs(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch transaction activity log
	changes := utils.InitChanges()

	utils.CompareChanges(oldAccount.Name, newAccount.Name, changes, "account")
	utils.CompareChanges(exTr.TransactionType, tr.TransactionType, changes, "type")
	utils.CompareDateChange(exTr.TxnDate, tr.TxnDate, changes, "date")
	utils.CompareDecimalChange(exTr.Amount, tr.Amount, changes, "amount", 2)
	utils.CompareChanges(exTr.Currency, tr.Currency, changes, "currency")
	utils.CompareChanges(oldCategory.Name, newCategory.Name, changes, "category")
	utils.CompareChanges(utils.SafeString(exTr.Description), utils.SafeString(tr.Description), changes, "description")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "transaction",
			Description: nil,
			Payload:     changes,
			Causer:      user,
		})
		if err != nil {
			return err
		}
	}

	// NEW account
	{
		newBal, err := s.AccountService.Repo.FindBalanceForAccountID(nil, newAccount.ID)
		if err != nil {
			return err
		}
		var delta decimal.Decimal
		if oldAccount.ID == newAccount.ID {
			delta = newEffect.Sub(oldEffect)
		} else {
			delta = newEffect
		}
		if !delta.IsZero() {
			start := newBal.EndBalance.Sub(delta)
			changes2 := utils.InitChanges()
			utils.CompareChanges("", newAccount.Name, changes2, "account")
			utils.CompareChanges("", delta.StringFixed(2), changes2, "change")
			utils.CompareChanges("", start.StringFixed(2), changes2, "start_balance")
			utils.CompareChanges("", newBal.EndBalance.StringFixed(2), changes2, "end_balance")
			utils.CompareChanges("", newAccount.Currency, changes2, "currency")

			if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
				LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
				Logger:      s.Ctx.Logger,
				Event:       "update",
				Category:    "balance",
				Description: nil,
				Payload:     changes2,
				Causer:      user,
			}); err != nil {
				return err
			}
		}
	}

	// OLD account (only if it changed)
	if oldAccount.ID != newAccount.ID && !oldEffect.IsZero() {
		oldBal, err := s.AccountService.Repo.FindBalanceForAccountID(nil, oldAccount.ID)
		if err != nil {
			return err
		}
		delta := oldEffect.Neg()
		start := oldBal.EndBalance.Sub(delta)
		changes3 := utils.InitChanges()
		utils.CompareChanges("", oldAccount.Name, changes3, "account")
		utils.CompareChanges("", delta.StringFixed(2), changes3, "change")
		utils.CompareChanges("", start.StringFixed(2), changes3, "start_balance")
		utils.CompareChanges("", oldBal.EndBalance.StringFixed(2), changes3, "end_balance")
		utils.CompareChanges("", oldAccount.Currency, changes3, "currency")

		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.LoggingRepo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "balance",
			Description: nil,
			Payload:     changes3,
			Causer:      user,
		}); err != nil {
			return err
		}
	}

	return nil
}
