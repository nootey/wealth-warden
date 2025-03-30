package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type SavingsService struct {
	SavingsRepo       *repositories.SavingsRepository
	AuthService       *AuthService
	LoggingService    *LoggingService
	RecActionsService *ReoccurringActionService
	Config            *config.Config
}

func NewSavingsService(
	cfg *config.Config,
	authService *AuthService,
	loggingService *LoggingService,
	recActionsService *ReoccurringActionService,
	repo *repositories.SavingsRepository,
) *SavingsService {
	return &SavingsService{
		SavingsRepo:       repo,
		AuthService:       authService,
		LoggingService:    loggingService,
		RecActionsService: recActionsService,
		Config:            cfg,
	}
}

func (s *SavingsService) FetchSavingsPaginated(c *gin.Context, paginationParams utils.PaginationParams, yearParam string) ([]models.SavingsAllocation, int, error) {

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

	totalRecords, err := s.SavingsRepo.CountSavings(user, year, paginationParams.Filters)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	records, err := s.SavingsRepo.FindSavings(user, year, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, paginationParams.Filters)
	if err != nil {
		return nil, 0, err
	}

	return records, int(totalRecords), nil
}

func (s *SavingsService) FetchAllSavingsGroupedByMonth(c *gin.Context, yearParam string) ([]models.SavingsSummary, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	currentYear := time.Now().Year()
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 {
		year = currentYear
	}

	var summaries []models.SavingsSummary

	total, err := s.SavingsRepo.FindTotalForGroupedSavings(user, year)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, total...)

	categorized, err := s.SavingsRepo.FetchGroupedSavingsByCategoryAndMonth(user, year)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, categorized...)

	sort.SliceStable(summaries, func(i, j int) bool {
		a, b := summaries[i], summaries[j]

		typeRank := map[string]int{
			"fixed":    0,
			"variable": 1,
		}

		// Compare category types (fixed before variable)
		if typeRank[a.CategoryType] != typeRank[b.CategoryType] {
			return typeRank[a.CategoryType] < typeRank[b.CategoryType]
		}

		// Inside "fixed" category, ensure "Total" (category_id == 0) comes first
		if a.CategoryType == "fixed" && b.CategoryType == "fixed" {
			if a.CategoryID == 0 && b.CategoryID != 0 {
				return true
			}
			if b.CategoryID == 0 && a.CategoryID != 0 {
				return false
			}
		}

		if a.CategoryID != b.CategoryID {
			return a.CategoryID < b.CategoryID
		}

		return a.Month < b.Month
	})

	return summaries, nil
}

func (s *SavingsService) FetchAllSavingsCategories(c *gin.Context) ([]models.SavingsCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.SavingsRepo.FindAllSavingCategories(user)
}

func (s *SavingsService) CreateSavingsAllocation(c *gin.Context, newRecord *models.SavingsAllocation) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.AllocatedAmount, 'f', 2, 64)
	dateStr := newRecord.AllocationDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", newRecord.SavingsCategory.Name, changes, "category")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", dateStr, changes, "allocation_date")

	newRecord.AdjustedAmount = &newRecord.AllocatedAmount

	err = s.SavingsRepo.InsertSavingsAllocation(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.SavingsRepo.UpdateCategoryGoalProgress(tx, user, newRecord, 1)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "savings_allocation", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *SavingsService) CreateSavingsDeduction(c *gin.Context, newRecord *models.SavingsDeduction) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.Amount, 'f', 2, 64)
	dateStr := newRecord.DeductionDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", newRecord.SavingsCategory.Name, changes, "category")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", dateStr, changes, "deduction_date")
	utils.CompareChanges("", utils.SafeString(newRecord.Reason), changes, "reason")

	err = s.SavingsRepo.InsertSavingsDeduction(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.SavingsRepo.UpdateCategoryGoalProgress(tx, user, newRecord.SavingsCategoryID, newRecord.Amount, -1)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "savings_deduction", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *SavingsService) CreateSavingsCategory(c *gin.Context, newRecord *models.SavingsCategory) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", newRecord.Name, changes, "category")
	utils.CompareChanges("", newRecord.SavingsType, changes, "type")
	utils.CompareChanges("", newRecord.AccountType, changes, "account_type")

	if newRecord.AccountType == "interest" && newRecord.InterestRate != nil {
		interestString := strconv.FormatFloat(*newRecord.InterestRate, 'f', 2, 64)
		utils.CompareChanges("", interestString, changes, "interest")
	}

	if newRecord.GoalTarget != nil {
		goalTargetString := strconv.FormatFloat(*newRecord.GoalTarget, 'f', 2, 64)
		utils.CompareChanges("", goalTargetString, changes, "goal_target")
	}

	err = s.SavingsRepo.InsertSavingsCategory(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "create", "savings_category", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *SavingsService) UpdateSavingsCategory(c *gin.Context, newRecord *models.SavingsCategory) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.SavingsRepo.GetSavingsCategoryByID(user, newRecord.ID)
	if err != nil {
		return err
	}

	utils.CompareChanges(existingRecord.Name, newRecord.Name, changes, "category")
	utils.CompareChanges(existingRecord.SavingsType, newRecord.SavingsType, changes, "type")
	utils.CompareChanges(existingRecord.AccountType, newRecord.AccountType, changes, "account_type")

	if newRecord.AccountType == "interest" && newRecord.InterestRate != nil {
		existingInterestString := strconv.FormatFloat(*existingRecord.InterestRate, 'f', 2, 64)
		interestString := strconv.FormatFloat(*newRecord.InterestRate, 'f', 2, 64)
		utils.CompareChanges(existingInterestString, interestString, changes, "interest")
	}

	if newRecord.GoalTarget != nil {
		existingGoalTargetString := strconv.FormatFloat(*existingRecord.GoalTarget, 'f', 2, 64)
		goalTargetString := strconv.FormatFloat(*newRecord.GoalTarget, 'f', 2, 64)
		utils.CompareChanges(existingGoalTargetString, goalTargetString, changes, "goal_target")
	}

	err = s.SavingsRepo.UpdateSavingsCategory(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	description := fmt.Sprintf("Savings category with ID: %d has been updated", newRecord.ID)

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "update", "savings_category", &description, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
