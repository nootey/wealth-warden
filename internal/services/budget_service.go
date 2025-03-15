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

	sums := make(map[string]float64)

	for _, mapping := range category.Mappings {
		inflowSums, err := s.FetchSumsForDynamicCategory("inflow", &mapping, year, month, user)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		outflowSums, err := s.FetchSumsForDynamicCategory("outflow", &mapping, year, month, user)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		sums["inflow"] += inflowSums
		sums["outflow"] += outflowSums
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

	ID, err := s.BudgetRepo.InsertBudget(tx, user, newRecord)
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
