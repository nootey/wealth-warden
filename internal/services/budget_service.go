package services

import (
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services/shared"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type BudgetService struct {
	BudgetRepo        *repositories.BudgetRepository
	AuthService       *AuthService
	LoggingService    *LoggingService
	RecActionsService *ReoccurringActionService
	BudgetInterface   *shared.BudgetInterface
	Config            *config.Config
}

func NewBudgetService(
	cfg *config.Config,
	authService *AuthService,
	loggingService *LoggingService,
	budgetInterface *shared.BudgetInterface,
	repo *repositories.BudgetRepository,
) *BudgetService {
	return &BudgetService{
		BudgetRepo:      repo,
		AuthService:     authService,
		LoggingService:  loggingService,
		BudgetInterface: budgetInterface,
		Config:          cfg,
	}
}

func (s *BudgetService) GetCurrentMonthlyBudget(c *gin.Context) (*models.MonthlyBudget, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	record, err := s.BudgetRepo.GetBudgetForMonth(user, year, month)
	if err != nil {
		return nil, err
	}

	// If no record is found, try for the previous month
	if record == nil {
		if month == 1 { // If January, go to December of the previous year
			year--
			month = 12
		} else {
			month--
		}

		record, err = s.BudgetRepo.GetBudgetForMonth(user, year, month)
		if err != nil {
			return nil, err
		}
	}

	return record, nil
}

func (s *BudgetService) FetchSumsForDynamicCategory(categoryName string, mapping *models.DynamicCategoryMapping, year, month int, user *models.User) (float64, error) {
	var sum float64
	var err error

	switch categoryName {
	case "inflow":
		{
			var inflowSums float64
			var dynamicSums float64

			if mapping.RelatedCategoryName == "inflow" {
				inflowSums, err = s.BudgetInterface.FetchTotalValuesForInflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			} else if mapping.RelatedCategoryName == "dynamic" {
				dynamicSums, err = s.BudgetInterface.FetchTotalValuesForDynamicCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			}

			sum = inflowSums + dynamicSums
		}
	case "outflow":
		{
			if mapping.RelatedCategoryName == "dynamic" {
				sum = 0
			} else {
				sum, err = s.BudgetInterface.FetchTotalValuesForOutflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			}
		}
	default:
		return 0, nil
	}

	return sum, nil
}

func (s *BudgetService) fetchSumsForBudget(category *models.DynamicCategory, user *models.User, year, month int) (map[string]float64, error) {
	sums := make(map[string]float64)

	for _, mapping := range category.Mappings {
		inflowSums, err := s.FetchSumsForDynamicCategory("inflow", &mapping, year, month, user)
		if err != nil {
			return nil, err
		}

		// Only sum outflows if it's the first time processing this category
		var totalOutflow float64
		if mapping.RelatedCategoryName == "outflow" {
			outflowSums, err := s.FetchSumsForDynamicCategory("outflow", &mapping, year, month, user)
			if err != nil {
				return nil, err
			}
			totalOutflow += outflowSums
		}

		sums["inflow"] += inflowSums
		sums["outflow"] += totalOutflow
	}

	return sums, nil
}

func (s *BudgetService) CreateMonthlyBudget(c *gin.Context, newRecord *models.MonthlyBudget) (*models.MonthlyBudget, error) {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	changes := utils.InitChanges()

	tx := s.BudgetRepo.Db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	category, err := s.BudgetInterface.InflowRepo.FindDynamicCategoryById(user, newRecord.DynamicCategoryID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	sums, err := s.fetchSumsForBudget(category, user, year, month)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	newRecord.Month = month
	newRecord.Year = year
	newRecord.TotalInflow = sums["inflow"]
	newRecord.TotalOutflow = sums["outflow"]
	newRecord.EffectiveBudget = newRecord.TotalInflow - newRecord.TotalOutflow
	newRecord.BudgetSnapshot = newRecord.TotalInflow - newRecord.TotalOutflow

	yearString := strconv.FormatInt(int64(year), 10)
	monthString := strconv.FormatInt(int64(month), 10)
	totalInflow := strconv.FormatInt(int64(newRecord.TotalInflow), 10)
	totalOutflow := strconv.FormatInt(int64(newRecord.TotalOutflow), 10)
	effectiveBudget := strconv.FormatInt(int64(newRecord.EffectiveBudget), 10)
	BudgetSnapshot := strconv.FormatInt(int64(newRecord.BudgetSnapshot), 10)

	utils.CompareChanges("", yearString, changes, "year")
	utils.CompareChanges("", monthString, changes, "month")
	utils.CompareChanges("", totalInflow, changes, "total_inflow")
	utils.CompareChanges("", totalOutflow, changes, "total_outflow")
	utils.CompareChanges("", effectiveBudget, changes, "effective_budget")
	utils.CompareChanges("", BudgetSnapshot, changes, "budget_snapshot")

	ID, err := s.BudgetRepo.InsertMonthlyBudget(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "monthly_budget", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	newRecord.ID = ID

	err = s.AuthService.UpdateBudgetInitializedStatus(tx, user, true)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return newRecord, tx.Commit().Error
}

func (s *BudgetService) CreateMonthlyBudgetAllocation(c *gin.Context, newRecord *models.MonthlyBudgetAllocation) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}

	budget, err := s.BudgetRepo.FindBudgetByID(newRecord.MonthlyBudgetID, user, true)
	if err != nil {
		return err
	}

	var totalAllocated float64

	for _, mapping := range budget.Allocations {
		totalAllocated += mapping.AllocatedValue
	}

	remaining := budget.BudgetSnapshot - totalAllocated

	if newRecord.AllocatedValue > remaining {
		return fmt.Errorf(
			"total budget allocation exceeds effective budget snapshot. "+
				"Max allowed allocation: €%.2f", remaining,
		)
	}

	changes := utils.InitChanges()

	tx := s.BudgetRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	yearString := strconv.FormatInt(int64(budget.Year), 10)
	monthString := strconv.FormatInt(int64(budget.Month), 10)
	allocationString := strconv.FormatInt(int64(newRecord.Allocation), 10)
	allocatedValueString := strconv.FormatInt(int64(newRecord.AllocatedValue), 10)

	utils.CompareChanges("", yearString, changes, "year")
	utils.CompareChanges("", monthString, changes, "month")
	utils.CompareChanges("", newRecord.Category, changes, "category")
	utils.CompareChanges("", newRecord.Method, changes, "method")
	utils.CompareChanges("", allocationString, changes, "allocation")
	utils.CompareChanges("", allocatedValueString, changes, "allocated_value")

	err = s.BudgetRepo.InsertMonthlyBudgetAllocation(tx, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "monthly_budget_allocation", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *BudgetService) handleBudgetAssignErrors(oldBudget *models.MonthlyBudget, newBudget *models.MonthlyBudgetUpdate) error {

	isSnapshotUpdated := newBudget.BudgetSnapshot != nil && *newBudget.BudgetSnapshot != oldBudget.BudgetSnapshot && newBudget.SnapshotThreshold == nil
	isThresholdUpdated := newBudget.SnapshotThreshold != nil && *newBudget.SnapshotThreshold != oldBudget.SnapshotThreshold && newBudget.BudgetSnapshot == nil

	// Validate BudgetSnapshot if it's being updated
	if isSnapshotUpdated {
		if *newBudget.BudgetSnapshot < 1 {
			return errors.New("snapshot must be a positive number")
		}
		if *newBudget.BudgetSnapshot > oldBudget.EffectiveBudget {
			return errors.New("snapshot cannot be higher than the effective budget")
		}
		if *newBudget.BudgetSnapshot < oldBudget.TotalOutflow {
			return errors.New("snapshot cannot be lower than the total outflows")
		}
	}

	// Validate SnapshotThreshold if it's being updated
	if isThresholdUpdated {
		if *newBudget.SnapshotThreshold < 1 {
			return errors.New("threshold must be a positive number")
		}
		if *newBudget.SnapshotThreshold > oldBudget.EffectiveBudget {
			return errors.New("threshold cannot exceed effective budget")
		}
		if *newBudget.SnapshotThreshold > oldBudget.BudgetSnapshot && oldBudget.BudgetSnapshot > 0 {
			return errors.New("threshold cannot exceed budget snapshot")
		}
	}

	return nil
}

func safeIntToString(value *float64) string {
	if value != nil {
		return strconv.FormatInt(int64(*value), 10)
	}
	return ""
}

func updateBudgetField(existingValue *float64, newValue *float64) {
	if newValue != nil {
		*existingValue = *newValue
	}
}

func (s *BudgetService) UpdateMonthlyBudget(c *gin.Context, newBudget *models.MonthlyBudgetUpdate) error {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.BudgetRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingBudget, err := s.BudgetRepo.FindBudgetByID(newBudget.ID, user, false)
	if err != nil {
		return err
	}

	err = s.handleBudgetAssignErrors(existingBudget, newBudget)
	if err != nil {
		return err
	}

	existingSnapshotString := strconv.FormatInt(int64(existingBudget.BudgetSnapshot), 10)
	existingThresholdString := strconv.FormatInt(int64(existingBudget.SnapshotThreshold), 10)

	newSnapshotString := safeIntToString(newBudget.BudgetSnapshot)
	newThresholdString := safeIntToString(newBudget.SnapshotThreshold)

	utils.CompareChanges(existingSnapshotString, newSnapshotString, changes, "budget_snapshot")
	utils.CompareChanges(existingThresholdString, newThresholdString, changes, "snapshot_threshold")

	if changes.HasChanges() {
		utils.CompareChanges("", strconv.Itoa(existingBudget.Month), changes, "month")
		utils.CompareChanges("", strconv.Itoa(existingBudget.Year), changes, "year")
	}

	updateBudgetField(&existingBudget.BudgetSnapshot, newBudget.BudgetSnapshot)
	updateBudgetField(&existingBudget.SnapshotThreshold, newBudget.SnapshotThreshold)

	err = s.BudgetRepo.UpdateMonthlyBudget(tx, user, existingBudget)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "monthly_budget", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *BudgetService) SynchronizeCurrentMonthlyBudget(c *gin.Context) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.BudgetRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	budget, err := s.BudgetRepo.GetBudgetForMonth(user, year, month)
	if err != nil {
		tx.Rollback()
		return err
	}

	existingTotalInflows := budget.TotalInflow
	existingTotalOutflows := budget.TotalOutflow
	existingEffectiveBudget := budget.EffectiveBudget
	existingBudgetSnapshot := budget.BudgetSnapshot

	existingTotalInflowsString := strconv.FormatInt(int64(existingTotalInflows), 10)
	existingTotalOutflowsString := strconv.FormatInt(int64(existingTotalOutflows), 10)
	existingEffectiveBudgetString := strconv.FormatInt(int64(existingEffectiveBudget), 10)
	existingBudgetSnapshotString := strconv.FormatInt(int64(existingBudgetSnapshot), 10)
	yearString := strconv.FormatInt(int64(budget.Year), 10)
	monthString := strconv.FormatInt(int64(budget.Month), 10)

	sums, err := s.fetchSumsForBudget(&budget.DynamicCategory, user, year, month)
	if err != nil {
		tx.Rollback()
		return err
	}

	budget.TotalInflow = sums["inflow"]
	budget.TotalOutflow = sums["outflow"]
	budget.EffectiveBudget = budget.TotalInflow - budget.TotalOutflow

	newEffectiveBudget := budget.EffectiveBudget

	if newEffectiveBudget != existingEffectiveBudget {

		budget.BudgetSnapshot = budget.EffectiveBudget
		newBudgetSnapshot := budget.BudgetSnapshot

		newEffectiveBudgetString := strconv.FormatInt(int64(newEffectiveBudget), 10)
		newBudgetSnapshotString := strconv.FormatInt(int64(newBudgetSnapshot), 10)
		newTotalInflowsString := strconv.FormatInt(int64(existingTotalInflows), 10)
		newTotalOutflowsString := strconv.FormatInt(int64(existingTotalOutflows), 10)

		utils.CompareChanges(existingEffectiveBudgetString, newEffectiveBudgetString, changes, "effective_budget")
		utils.CompareChanges(existingBudgetSnapshotString, newBudgetSnapshotString, changes, "budget_snapshot")
		utils.CompareChanges(existingTotalInflowsString, newTotalInflowsString, changes, "total_inflows")
		utils.CompareChanges(existingTotalOutflowsString, newTotalOutflowsString, changes, "total_outflows")
		utils.CompareChanges("", monthString, changes, "month")
		utils.CompareChanges("", yearString, changes, "year")

		err = s.BudgetRepo.UpdateMonthlyBudget(tx, user, budget)
		if err != nil {
			tx.Rollback()
			return err
		}

		description := "User has synchronized their monthly budget. Some values were out of sync"

		err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "sync", "monthly_budget", &description, changes, user)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *BudgetService) SynchronizeCurrentMonthlyBudgetSnapshot(c *gin.Context) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.BudgetRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	budget, err := s.BudgetRepo.GetBudgetForMonth(user, year, month)
	if err != nil {
		tx.Rollback()
		return err
	}

	existingBudgetSnapshot := budget.BudgetSnapshot

	existingBudgetSnapshotString := strconv.FormatInt(int64(existingBudgetSnapshot), 10)
	yearString := strconv.FormatInt(int64(budget.Year), 10)
	monthString := strconv.FormatInt(int64(budget.Month), 10)

	budget.BudgetSnapshot = budget.EffectiveBudget

	if existingBudgetSnapshot != budget.BudgetSnapshot {

		newBudgetSnapshot := budget.BudgetSnapshot

		newBudgetSnapshotString := strconv.FormatInt(int64(newBudgetSnapshot), 10)

		utils.CompareChanges(existingBudgetSnapshotString, newBudgetSnapshotString, changes, "budget_snapshot")
		utils.CompareChanges("", monthString, changes, "month")
		utils.CompareChanges("", yearString, changes, "year")

		err = s.BudgetRepo.UpdateMonthlyBudget(tx, user, budget)
		if err != nil {
			tx.Rollback()
			return err
		}

		description := "User has synchronized their monthly budget snapshot. Some values were out of sync"

		err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "sync", "monthly_budget_snapshot", &description, changes, user)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
