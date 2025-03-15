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

func (s *OutflowService) FetchOutflowsPaginated(c *gin.Context, paginationParams utils.PaginationParams, yearParam string) ([]models.Outflow, int, error) {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, 0, err
	}

	// Get the current year
	currentYear := time.Now().Year()

	// Convert yearParam to integer
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 { // Ensure year is valid
		year = currentYear // Default to current year if invalid
	}

	totalRecords, err := s.OutflowRepo.CountOutflows(user, year)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	outflows, err := s.OutflowRepo.FindOutflows(user, year, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	return outflows, int(totalRecords), nil
}

func (s *OutflowService) FetchAllOutflowsGroupedByMonth(c *gin.Context, yearParam string) ([]models.OutflowSummary, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	currentYear := time.Now().Year()
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 {
		year = currentYear
	}

	return s.OutflowRepo.FindAllOutflowsGroupedByMonth(user, year)
}

func (s *OutflowService) FetchAllOutflowCategories(c *gin.Context) ([]models.OutflowCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.OutflowRepo.GetAllOutflowCategories(user)
}

func (s *OutflowService) CreateOutflow(c *gin.Context, newRecord *models.Outflow) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	outflowDateStr := newRecord.OutflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", outflowDateStr, changes, "outflow_date")
	utils.CompareChanges("", newRecord.OutflowCategory.Name, changes, "outflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.OutflowRepo.InsertOutflow(tx, user, newRecord)
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

func (s *OutflowService) UpdateOutflow(c *gin.Context, newRecord *models.Outflow) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.OutflowRepo.GetOutflowByID(user, newRecord.ID)
	if err != nil {
		return err
	}

	existingAmountString := strconv.FormatFloat(existingRecord.Amount, 'f', 2, 64)
	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	existingOutflowDateStr := existingRecord.OutflowDate.UTC().Format(time.RFC3339)
	outflowDateStr := newRecord.OutflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges(existingOutflowDateStr, outflowDateStr, changes, "outflow_date")
	utils.CompareChanges(existingRecord.OutflowCategory.Name, newRecord.OutflowCategory.Name, changes, "outflow_category")
	utils.CompareChanges(existingAmountString, amountString, changes, "amount")
	utils.CompareChanges(utils.SafeString(existingRecord.Description), utils.SafeString(newRecord.Description), changes, "description")

	_, err = s.OutflowRepo.UpdateOutflow(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	description := fmt.Sprintf("Updated record with ID: %d", newRecord.ID)

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "outflow", &description, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) CreateReoccurringOutflow(c *gin.Context, newRecord *models.Outflow, newReoccurringRecord *models.RecurringAction) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	outflowDateStr := newRecord.OutflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", outflowDateStr, changes, "outflow_date")
	utils.CompareChanges("", newRecord.OutflowCategory.Name, changes, "outflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.OutflowRepo.InsertOutflow(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	startDateStr := newReoccurringRecord.StartDate.UTC().Format(time.RFC3339)
	var endDateStr *string
	if newReoccurringRecord.EndDate != nil {
		formatted := newReoccurringRecord.EndDate.UTC().Format(time.RFC3339)
		endDateStr = &formatted
	} else {
		endDateStr = nil // Ensure it remains nil instead of an empty string
	}

	if endDateStr != nil {
		utils.CompareChanges("", *endDateStr, changes, "end_date")
	}

	utils.CompareChanges("", startDateStr, changes, "start_date")
	utils.CompareChanges("", newReoccurringRecord.CategoryType, changes, "category")
	utils.CompareChanges("", newReoccurringRecord.IntervalUnit, changes, "interval_unit")
	utils.CompareChanges("", strconv.Itoa(newReoccurringRecord.IntervalValue), changes, "interval_value")

	_, err = s.RecActionsService.ActionRepo.InsertReoccurringAction(tx, user, newReoccurringRecord)
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

func (s *OutflowService) CreateOutflowCategory(c *gin.Context, newRecord *models.OutflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", newRecord.Name, changes, "category")

	err = s.OutflowRepo.InsertOutflowCategory(tx, user, newRecord)
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

func (s *OutflowService) UpdateOutflowCategory(c *gin.Context, newRecord *models.OutflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.OutflowRepo.GetOutflowCategoryByID(user, newRecord.ID)
	if err != nil {
		return err
	}

	utils.CompareChanges(existingRecord.Name, newRecord.Name, changes, "category")

	err = s.OutflowRepo.UpdateOutflowCategory(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	description := fmt.Sprintf("Outflow category with ID: %d has been updated", newRecord.ID)

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "outflow_category", &description, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *OutflowService) DeleteOutflow(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.OutflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	outflow, err := s.OutflowRepo.GetOutflowByID(user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	amountString := strconv.FormatFloat(outflow.Amount, 'f', 2, 64)

	utils.CompareChanges(outflow.OutflowCategory.Name, "", changes, "outflow")
	utils.CompareChanges(amountString, "", changes, "amount")

	err = s.OutflowRepo.DropOutflow(tx, user, id)
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

	user, err := s.AuthService.GetCurrentUser(c, false)
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

	OutflowCategory, err := s.OutflowRepo.GetOutflowCategoryByID(user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.OutflowRepo.CountOutflowsByCategory(user, id, &CountOutflows); err != nil {
		return err
	}
	if err := s.RecActionsService.ActionRepo.CountReoccurringActionByCategory(user, "outflow", id, &countActions); err != nil {
		return err
	}

	if CountOutflows > 0 {
		return fmt.Errorf("cannot delete outflow category: it is being used by %d outflow(s)", CountOutflows)
	}

	if countActions > 0 {
		return fmt.Errorf("cannot delete outflow category: it is being used by %d reoccurring action(s)", countActions)
	}

	utils.CompareChanges(OutflowCategory.Name, "", changes, "category")

	err = s.OutflowRepo.DropOutflowCategory(tx, user, id)
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
