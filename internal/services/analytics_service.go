package services

import (
	"wealth-warden/internal/repositories"
)

type AnalyticsServiceInterface interface {
}
type AnalyticsService struct {
	repo         repositories.AnalyticsRepositoryInterface
	accRepo      repositories.AccountRepositoryInterface
	txnRepo      repositories.TransactionRepositoryInterface
	settingsRepo repositories.SettingsRepositoryInterface
}

func NewAnalyticsService(
	repo *repositories.AnalyticsRepository,
	accRepo *repositories.AccountRepository,
	txRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
) *AnalyticsService {
	return &AnalyticsService{
		repo:         repo,
		accRepo:      accRepo,
		txnRepo:      txRepo,
		settingsRepo: settingsRepo,
	}
}

var _ AnalyticsServiceInterface = (*AnalyticsService)(nil)
