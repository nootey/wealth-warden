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

func (s *InflowService) CreateInflow(c *gin.Context, newRecord *models.Inflow) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	inflowDateStr := newRecord.InflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", inflowDateStr, changes, "inflow_date")
	utils.CompareChanges("", newRecord.InflowCategory.Name, changes, "inflow_category")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", utils.SafeString(newRecord.Description), changes, "description")

	_, err = s.InflowRepo.InsertInflow(tx, user.ID, newRecord)
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

func (s *InflowService) UpdateInflow(c *gin.Context, newRecord *models.Inflow) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.InflowRepo.GetInflowByID(user.ID, newRecord.ID)
	if err != nil {
		return err
	}

	existingAmountString := strconv.FormatFloat(existingRecord.Amount, 'f', 2, 64)
	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	existingInflowDateStr := existingRecord.InflowDate.UTC().Format(time.RFC3339)
	inflowDateStr := newRecord.InflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges(existingInflowDateStr, inflowDateStr, changes, "inflow_date")
	utils.CompareChanges(existingRecord.InflowCategory.Name, newRecord.InflowCategory.Name, changes, "inflow_category")
	utils.CompareChanges(existingAmountString, amountString, changes, "amount")
	utils.CompareChanges(utils.SafeString(existingRecord.Description), utils.SafeString(newRecord.Description), changes, "description")

	_, err = s.InflowRepo.UpdateInflow(tx, user.ID, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	description := fmt.Sprintf("Updated record with ID: %d", newRecord.ID)

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "inflow", &description, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *InflowService) CreateReoccurringInflow(c *gin.Context, newRecord *models.Inflow, newReoccurringRecord *models.RecurringAction) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	inflowDateStr := newRecord.InflowDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", inflowDateStr, changes, "inflow_date")
	utils.CompareChanges("", newRecord.InflowCategory.Name, changes, "inflow_category")
	utils.CompareChanges("", amountString, changes, "amount")

	_, err = s.InflowRepo.InsertInflow(tx, user.ID, newRecord)
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

	_, err = s.RecActionsService.ActionRepo.InsertReoccurringAction(tx, user.ID, newReoccurringRecord)
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

func (s *InflowService) CreateInflowCategory(c *gin.Context, newRecord *models.InflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", newRecord.Name, changes, "category")

	err = s.InflowRepo.InsertInflowCategory(tx, user.ID, newRecord)
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

func (s *InflowService) UpdateInflowCategory(c *gin.Context, newRecord *models.InflowCategory) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.InflowRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.InflowRepo.GetInflowCategoryByID(user.ID, newRecord.ID)
	if err != nil {
		return err
	}

	utils.CompareChanges(existingRecord.Name, newRecord.Name, changes, "category")

	err = s.InflowRepo.UpdateInflowCategory(tx, user.ID, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	description := fmt.Sprintf("Inflow category with ID: %d has been updated", newRecord.ID)

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "inflow_category", &description, changes, user)
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
	var countInflows int64
	var countActions int64
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

	if err := s.InflowRepo.CountInflowsByCategory(user.ID, id, &countInflows); err != nil {
		return err
	}
	if err := s.RecActionsService.ActionRepo.CountReoccurringActionByCategory(user.ID, "inflow", id, &countActions); err != nil {
		return err
	}

	if countInflows > 0 {
		return fmt.Errorf("cannot delete inflow category: it is being used by %d inflow(s)", countInflows)
	}

	if countActions > 0 {
		return fmt.Errorf("cannot delete inflow category: it is being used by %d reoccurring action(s)", countActions)
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
