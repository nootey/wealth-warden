package services

import (
	"github.com/gin-gonic/gin"
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

func (s *SavingsService) FetchAllSavingsCategories(c *gin.Context) ([]models.SavingsCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.SavingsRepo.FindAllSavingCategories(user)
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

	if newRecord.GoalValue != nil {
		goalValueString := strconv.FormatFloat(*newRecord.GoalValue, 'f', 2, 64)
		utils.CompareChanges("", goalValueString, changes, "goal_value")
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
