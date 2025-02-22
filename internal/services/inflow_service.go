package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type InflowService struct {
	InflowRepo        *repositories.InflowRepository
	AuthService       *AuthService
	LoggingService    *LoggingService
	RecActionsService *ReoccurringActionService
	Config            *config.Config
}

func NewInflowService(
	cfg *config.Config,
	authService *AuthService,
	loggingService *LoggingService,
	recActionsService *ReoccurringActionService,
	repo *repositories.InflowRepository,
) *InflowService {
	return &InflowService{
		InflowRepo:        repo,
		AuthService:       authService,
		LoggingService:    loggingService,
		RecActionsService: recActionsService,
		Config:            cfg,
	}
}

func (s *InflowService) FetchInflowsPaginated(c *gin.Context, paginationParams utils.PaginationParams) ([]models.Inflow, int, error) {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, 0, err
	}

	totalRecords, err := s.InflowRepo.CountInflows(user.ID)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	inflows, err := s.InflowRepo.FindInflows(user.ID, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	return inflows, int(totalRecords), nil
}

func (s *InflowService) FetchAllInflowsGroupedByMonth(c *gin.Context) ([]models.InflowSummary, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.InflowRepo.FindAllInflowsGroupedByMonth(user.ID)
}

func (s *InflowService) FetchAllInflowCategories(c *gin.Context) ([]models.InflowCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.InflowRepo.GetAllInflowCategories(user.ID)
}

func (s *InflowService) CreateInflow(c *gin.Context, inflow *models.Inflow) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(inflow.Amount, 'f', 2, 64)
	inflowDateStr := inflow.InflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", inflowDateStr, changes, "inflow_date")
	utils.CompareChanges("", inflow.InflowCategory.Name, changes, "inflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.InflowRepo.InsertInflow(tx, user.ID, inflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "inflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InflowService) CreateReoccurringInflow(c *gin.Context, inflow *models.Inflow, reoccurringInflow *models.RecurringAction) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(inflow.Amount, 'f', 2, 64)
	inflowDateStr := inflow.InflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", inflowDateStr, changes, "inflow_date")
	utils.CompareChanges("", inflow.InflowCategory.Name, changes, "inflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	inflowID, err := s.InflowRepo.InsertInflow(tx, user.ID, inflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	reoccurringInflow.CategoryID = inflowID

	startDateStr := reoccurringInflow.StartDate.UTC().Format(time.RFC3339)
	var endDateStr *string
	if reoccurringInflow.EndDate != nil {
		formatted := reoccurringInflow.EndDate.UTC().Format(time.RFC3339)
		endDateStr = &formatted
	} else {
		endDateStr = nil // Ensure it remains nil instead of an empty string
	}

	if endDateStr != nil {
		utils.CompareChanges("", *endDateStr, changes, "end_date")
	}

	utils.CompareChanges("", startDateStr, changes, "start_date")
	utils.CompareChanges("", reoccurringInflow.CategoryType, changes, "category")
	utils.CompareChanges("", reoccurringInflow.IntervalUnit, changes, "interval_unit")
	utils.CompareChanges("", strconv.Itoa(reoccurringInflow.IntervalValue), changes, "interval_value")

	_, err = s.RecActionsService.ActionRepo.InsertReoccurringAction(tx, user.ID, reoccurringInflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "reoccurring-inflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InflowService) CreateInflowCategory(c *gin.Context, inflowCategory *models.InflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", inflowCategory.Name, changes, "category")

	err = s.InflowRepo.InsertInflowCategory(tx, user.ID, inflowCategory)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "inflow_category", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InflowService) DeleteInflow(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	inflow, err := s.InflowRepo.GetInflowByID(user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	amountString := strconv.FormatFloat(inflow.Amount, 'f', 2, 64)

	utils.CompareChanges(inflow.InflowCategory.Name, "", changes, "inflow")
	utils.CompareChanges(amountString, "", changes, "amount")

	err = s.InflowRepo.DropInflow(tx, user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "delete", "inflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InflowService) DeleteInflowCategory(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	var count int64
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	inflowCategory, err := s.InflowRepo.GetInflowCategoryByID(user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.InflowRepo.CountInflowsByCategory(user.ID, id, &count); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete inflow category: it is being used by %d inflow(s)", count)
	}

	utils.CompareChanges(inflowCategory.Name, "", changes, "category")

	err = s.InflowRepo.DropInflowCategory(tx, user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "delete", "inflow_category", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
