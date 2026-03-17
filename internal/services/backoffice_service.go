package services

import (
	"wealth-warden/internal/repositories"
)

type BackofficeServiceInterface interface {
}
type BackofficeService struct {
	repo         repositories.BackofficeRepositoryInterface
	settingsRepo repositories.SettingsRepositoryInterface
}

func NewBackofficeService(
	repo *repositories.BackofficeRepository,
	settingsRepo *repositories.SettingsRepository,
) *BackofficeService {
	return &BackofficeService{
		repo:         repo,
		settingsRepo: settingsRepo,
	}
}

var _ BackofficeServiceInterface = (*BackofficeService)(nil)
