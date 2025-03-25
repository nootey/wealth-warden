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
