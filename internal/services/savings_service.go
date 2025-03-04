package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
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
