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

type OutflowService struct {
	OutflowRepo       *repositories.OutflowRepository
	AuthService       *AuthService
	LoggingService    *LoggingService
	RecActionsService *ReoccurringActionService
	Config            *config.Config
}

func NewOutflowService(
	cfg *config.Config,
	authService *AuthService,
	loggingService *LoggingService,
	recActionsService *ReoccurringActionService,
	repo *repositories.OutflowRepository,
) *OutflowService {
	return &OutflowService{
		OutflowRepo:       repo,
		AuthService:       authService,
		LoggingService:    loggingService,
		RecActionsService: recActionsService,
		Config:            cfg,
	}
}

func (s *OutflowService) FetchOutflowsPaginated(c *gin.Context, paginationParams utils.PaginationParams) ([]models.Outflow, int, error) {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, 0, err
	}

	totalRecords, err := s.OutflowRepo.CountOutflows(user.ID)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	outflows, err := s.OutflowRepo.FindOutflows(user.ID, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	return outflows, int(totalRecords), nil
}

func (s *OutflowService) FetchAllOutflowsGroupedByMonth(c *gin.Context) ([]models.OutflowSummary, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.OutflowRepo.FindAllOutflowsGroupedByMonth(user.ID)
}

func (s *OutflowService) FetchAllOutflowCategories(c *gin.Context) ([]models.OutflowCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.OutflowRepo.GetAllOutflowCategories(user.ID)
}

func (s *OutflowService) CreateOutflow(c *gin.Context, outflow *models.Outflow) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(outflow.Amount, 'f', 2, 64)
	outflowDateStr := outflow.OutflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", outflowDateStr, changes, "outflow_date")
	utils.CompareChanges("", outflow.OutflowCategory.Name, changes, "outflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.OutflowRepo.InsertOutflow(tx, user.ID, outflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "outflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) CreateReoccurringOutflow(c *gin.Context, outflow *models.Outflow, reoccurringOutflow *models.RecurringAction) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(outflow.Amount, 'f', 2, 64)
	outflowDateStr := outflow.OutflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", outflowDateStr, changes, "outflow_date")
	utils.CompareChanges("", outflow.OutflowCategory.Name, changes, "outflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.OutflowRepo.InsertOutflow(tx, user.ID, outflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	startDateStr := reoccurringOutflow.StartDate.UTC().Format(time.RFC3339)
	var endDateStr *string
	if reoccurringOutflow.EndDate != nil {
		formatted := reoccurringOutflow.EndDate.UTC().Format(time.RFC3339)
		endDateStr = &formatted
	} else {
		endDateStr = nil // Ensure it remains nil instead of an empty string
	}

	if endDateStr != nil {
		utils.CompareChanges("", *endDateStr, changes, "end_date")
	}

	utils.CompareChanges("", startDateStr, changes, "start_date")
	utils.CompareChanges("", reoccurringOutflow.CategoryType, changes, "category")
	utils.CompareChanges("", reoccurringOutflow.IntervalUnit, changes, "interval_unit")
	utils.CompareChanges("", strconv.Itoa(reoccurringOutflow.IntervalValue), changes, "interval_value")

	_, err = s.RecActionsService.ActionRepo.InsertReoccurringAction(tx, user.ID, reoccurringOutflow)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "reoccurring-outflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) CreateOutflowCategory(c *gin.Context, OutflowCategory *models.OutflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", OutflowCategory.Name, changes, "category")

	err = s.OutflowRepo.InsertOutflowCategory(tx, user.ID, OutflowCategory)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "outflow_category", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) DeleteOutflow(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	outflow, err := s.OutflowRepo.GetOutflowByID(user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	amountString := strconv.FormatFloat(outflow.Amount, 'f', 2, 64)

	utils.CompareChanges(outflow.OutflowCategory.Name, "", changes, "outflow")
	utils.CompareChanges(amountString, "", changes, "amount")

	err = s.OutflowRepo.DropOutflow(tx, user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "delete", "outflow", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) DeleteOutflowCategory(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	var CountOutflows int64
	var countActions int64
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	OutflowCategory, err := s.OutflowRepo.GetOutflowCategoryByID(user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.OutflowRepo.CountOutflowsByCategory(user.ID, id, &CountOutflows); err != nil {
		return err
	}
	if err := s.RecActionsService.ActionRepo.CountReoccurringActionByCategory(user.ID, "outflow", id, &countActions); err != nil {
		return err
	}

	if CountOutflows > 0 {
		return fmt.Errorf("cannot delete outflow category: it is being used by %d outflow(s)", CountOutflows)
	}

	if countActions > 0 {
		return fmt.Errorf("cannot delete outflow category: it is being used by %d reoccurring action(s)", countActions)
	}

	utils.CompareChanges(OutflowCategory.Name, "", changes, "category")

	err = s.OutflowRepo.DropOutflowCategory(tx, user.ID, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "delete", "outflow_category", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
