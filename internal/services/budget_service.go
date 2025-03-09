package services

import (
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
