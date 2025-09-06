package services

import (
	"errors"
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

func (s *TransactionService) FetchTransactionsPaginated(c *gin.Context, includeDeleted bool, accountID *int64) ([]models.Transaction, *utils.Paginator, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, nil, err
	}

	queryParams := c.Request.URL.Query()
	p := utils.GetPaginationParams(queryParams)

	totalRecords, err := s.Repo.CountTransactions(user, p.Filters, includeDeleted, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindTransactions(user, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeDeleted, accountID)
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

func (s *TransactionService) FetchTransfersPaginated(c *gin.Context, includeDeleted bool) ([]models.Transfer, *utils.Paginator, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, nil, err
	}

	queryParams := c.Request.URL.Query()
	p := utils.GetPaginationParams(queryParams)

	totalRecords, err := s.Repo.CountTransfers(user, includeDeleted)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindTransfers(user, offset, p.RowsPerPage, includeDeleted)
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

func (s *TransactionService) FetchTransactionByID(c *gin.Context, id int64, includeDeleted bool) (*models.Transaction, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}

	record, err := s.Repo.FindTransactionByID(nil, id, user.ID, includeDeleted)
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

	account, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, user.ID, false)
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

	_, err = s.Repo.InsertTransaction(tx, &tr)
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
		LoggingRepo: s.Ctx.LoggingService.Repo,
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
	var change decimal.Decimal
	if tr.TransactionType == "expense" {
		change = tr.Amount.Neg()
	} else {
		change = tr.Amount
	}

	if err := s.AccountService.LogBalanceChange(&account, user, change); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) InsertTransfer(c *gin.Context, req *models.TransferReq) error {

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

	fromAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.SourceID, user.ID, true)
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

	toAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.DestinationID, user.ID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find destination account %w", err)
	}

	outflow := models.Transaction{
		UserID:          user.ID,
		AccountID:       fromAccount.ID,
		TransactionType: "expense",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         time.Now(),
		Description:     req.Notes,
	}

	if _, err := s.Repo.InsertTransaction(tx, &outflow); err != nil {
		tx.Rollback()
		return err
	}

	inflow := models.Transaction{
		UserID:          user.ID,
		AccountID:       toAccount.ID,
		TransactionType: "income",
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         time.Now(),
		Description:     req.Notes,
	}

	if _, err := s.Repo.InsertTransaction(tx, &inflow); err != nil {
		tx.Rollback()
		return err
	}

	transfer := models.Transfer{
		UserID:               user.ID,
		TransactionInflowID:  inflow.ID,
		TransactionOutflowID: outflow.ID,
		Amount:               req.Amount,
		Currency:             models.DefaultCurrency,
		Status:               "success",
		Notes:                req.Notes,
	}

	if _, err := s.Repo.InsertTransfer(tx, &transfer); err != nil {
		tx.Rollback()
		return err
	}

	// Update balances
	if err := s.AccountService.UpdateAccountCashBalance(tx, &fromAccount, "expense", req.Amount); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.AccountService.UpdateAccountCashBalance(tx, &toAccount, "income", req.Amount); err != nil {
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
		Causer:      user,
	}); err != nil {
		return err
	}

	// Log balance updates for both accounts
	if err := s.AccountService.LogBalanceChange(&fromAccount, user, req.Amount.Neg()); err != nil {
		return err
	}
	if err := s.AccountService.LogBalanceChange(&toAccount, user, req.Amount); err != nil {
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

	exTr, err := s.Repo.FindTransactionByID(tx, id, user.ID, false)
	if err != nil {
		return fmt.Errorf("can't find transaction with given id %w", err)
	}

	if exTr.IsAdjustment {
		return errors.New("can't edit a manual adjustment transaction")
	}

	// Load existing relations for comparison
	oldAccount, err := s.AccountService.Repo.FindAccountByID(tx, exTr.AccountID, user.ID, false)
	if err != nil {
		return fmt.Errorf("can't find existing account: %w", err)
	}
	var oldCategory models.Category
	if exTr.CategoryID != nil {
		oldCategory, _ = s.Repo.FindCategoryByID(tx, *exTr.CategoryID, &user.ID)
	}

	// Resolve new relations  from req
	newAccount, err := s.AccountService.Repo.FindAccountByID(tx, req.AccountID, user.ID, false)
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
	utils.CompareDateChange(&exTr.TxnDate, &tr.TxnDate, changes, "date")
	utils.CompareDecimalChange(&exTr.Amount, &tr.Amount, changes, "amount", 2)
	utils.CompareChanges(exTr.Currency, tr.Currency, changes, "currency")
	utils.CompareChanges(oldCategory.Name, newCategory.Name, changes, "category")
	utils.CompareChanges(utils.SafeString(exTr.Description), utils.SafeString(tr.Description), changes, "description")

	if !changes.IsEmpty() {
		err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
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
		var delta decimal.Decimal
		if oldAccount.ID == newAccount.ID {
			delta = newEffect.Sub(oldEffect)
		} else {
			delta = newEffect
		}
		if !delta.IsZero() {
			if err := s.AccountService.LogBalanceChange(&newAccount, user, delta); err != nil {
				return err
			}
		}
	}

	// OLD account (only if it changed)
	if oldAccount.ID != newAccount.ID && !oldEffect.IsZero() {
		if err := s.AccountService.LogBalanceChange(&oldAccount, user, oldEffect.Neg()); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteTransaction(c *gin.Context, id int64) error {

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

	// Load the transaction + relations
	tr, err := s.Repo.FindTransactionByID(tx, id, user.ID, false)
	if err != nil {
		return fmt.Errorf("can't find transaction with given id %w", err)
	}

	account, err := s.AccountService.Repo.FindAccountByID(tx, tr.AccountID, user.ID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find account with given id %w", err)
	}

	var category models.Category
	if tr.CategoryID != nil {
		cat, err := s.Repo.FindCategoryByID(tx, *tr.CategoryID, &user.ID)
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
		if err := s.AccountService.UpdateAccountCashBalance(tx, &account, dir, inverse.Abs()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Delete transaction
	if err := s.Repo.DeleteTransaction(tx, tr.ID, user.ID); err != nil {
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
			Causer:      user,
		})
		if err != nil {
			return err
		}
	}

	// Dispatch balance change on the affected account activity log
	if !inverse.IsZero() {
		if err := s.AccountService.LogBalanceChange(&account, user, inverse); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionService) DeleteTransfer(c *gin.Context, id int64) error {

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

	// Load the transfer
	transfer, err := s.Repo.FindTransferByID(tx, id, user.ID)
	if err != nil {
		return fmt.Errorf("can't find transfer with given id %w", err)
	}

	// Load associated transactions
	inflow, err := s.Repo.FindTransactionByID(tx, transfer.TransactionInflowID, user.ID, false)
	if err != nil {
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}

	outflow, err := s.Repo.FindTransactionByID(tx, transfer.TransactionOutflowID, user.ID, false)
	if err != nil {
		return fmt.Errorf("can't find outflow transaction with given id %w", err)
	}

	// Load accounts
	fromAcc, err := s.AccountService.Repo.FindAccountByID(tx, outflow.AccountID, user.ID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}
	toAcc, err := s.AccountService.Repo.FindAccountByID(tx, inflow.AccountID, user.ID, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find destination account %w", err)
	}

	// Reverse balances
	if err := s.AccountService.UpdateAccountCashBalance(tx, &fromAcc, "income", outflow.Amount); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.AccountService.UpdateAccountCashBalance(tx, &toAcc, "expense", inflow.Amount); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transfer
	if err := s.Repo.DeleteTransfer(tx, transfer.ID, user.ID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete transactions
	if err := s.Repo.DeleteTransaction(tx, inflow.ID, user.ID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.Repo.DeleteTransaction(tx, outflow.ID, user.ID); err != nil {
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
			Causer:      user,
		}); err != nil {
			return err
		}
	}

	// Log balance changes
	if err := s.AccountService.LogBalanceChange(&fromAcc, user, outflow.Amount); err != nil {
		return err
	}
	if err := s.AccountService.LogBalanceChange(&toAcc, user, inflow.Amount.Neg()); err != nil {
		return err
	}

	return nil
}

func (s *TransactionService) RestoreTransaction(c *gin.Context, id int64) error {

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

	// Load the transaction
	tr, err := s.Repo.FindTransactionByID(tx, id, user.ID, true)
	if err != nil {
		return fmt.Errorf("can't find inflow transaction with given id %w", err)
	}
	if tr.DeletedAt == nil {
		tx.Rollback()
		return fmt.Errorf("transaction is not deleted")
	}

	// Load account
	acc, err := s.AccountService.Repo.FindAccountByID(tx, tr.AccountID, user.ID, false)
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
		if err := s.AccountService.UpdateAccountCashBalance(tx, &acc, dir, origEffect.Abs()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Unmark as soft deleted
	if err := s.Repo.RestoreTransaction(tx, tr.ID, user.ID); err != nil {
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
		Causer:      user,
	}); err != nil {
		return err
	}

	// Log balance changes
	if !origEffect.IsZero() {
		if err := s.AccountService.LogBalanceChange(&acc, user, origEffect); err != nil {
			return err
		}
	}

	return nil
}
