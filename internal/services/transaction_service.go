package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
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
	yearParam := queryParams.Get("year")

	// Get the current year
	currentYear := time.Now().Year()

	// Convert yearParam to integer
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 { // Ensure year is valid
		year = currentYear // Default to current year if invalid
	}

	totalRecords, err := s.Repo.CountTransactions(user, year, paginationParams.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	records, err := s.Repo.FindTransactions(user, year, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, paginationParams.Filters)
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

func (s *TransactionService) FetchAllCategories(c *gin.Context) ([]models.Category, error) {
	return s.Repo.FindAllCategories(nil)
}

func (s *TransactionService) InsertTransaction(c *gin.Context, req *models.TransactionCreateReq) error {

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

	account, err := s.AccountService.Repo.FindAccountByID(req.AccountID, user.ID)
	if err != nil {
		return fmt.Errorf("can't find account with given id %w", err)
	}

	var category models.Category
	if req.CategoryID != nil {
		category, err = s.Repo.FindCategoryByID(*req.CategoryID, &user.ID)
		if err != nil {
			return fmt.Errorf("can't find category with given id %w", err)
		}
	}

	tr := models.Transaction{
		UserID:          user.ID,
		AccountID:       account.ID,
		TransactionType: strings.ToLower(req.TransactionType),
		Amount:          req.Amount,
		Currency:        models.DefaultCurrency,
		TxnDate:         req.TxnDate,
		Description:     req.Description,
	}

	if category.ID != 0 {
		tr.CategoryID = &category.ID
	}

	changes := utils.InitChanges()
	amountString := strconv.FormatFloat(tr.Amount, 'f', 2, 64)
	dateStr := tr.TxnDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", tr.TransactionType, changes, "type")
	utils.CompareChanges("", dateStr, changes, "date")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", tr.Currency, changes, "currency")

	if tr.CategoryID != nil {
		utils.CompareChanges("", category.Name, changes, "category")
	}

	utils.CompareChanges("", utils.SafeString(tr.Description), changes, "description")

	_, err = s.Repo.InsertTransaction(tx, tr)
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
		Category:    "transaction",
		Description: nil,
		Payload:     changes,
		Causer:      user,
	})
	if err != nil {
		return err
	}

	return nil
}
