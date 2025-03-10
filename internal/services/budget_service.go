package services

import (
	"github.com/gin-gonic/gin"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type BudgetService struct {
	BudgetRepo        *repositories.BudgetRepository
	AuthService       *AuthService
	LoggingService    *LoggingService
	RecActionsService *ReoccurringActionService
	Config            *config.Config
}

func NewBudgetService(
	cfg *config.Config,
	authService *AuthService,
	loggingService *LoggingService,
	repo *repositories.BudgetRepository,
) *BudgetService {
	return &BudgetService{
		BudgetRepo:     repo,
		AuthService:    authService,
		LoggingService: loggingService,
		Config:         cfg,
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
