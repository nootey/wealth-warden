package services

import (
	"fmt"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type SettingsService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.SettingsRepository
}

func NewSettingsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.SettingsRepository,
) *SettingsService {
	return &SettingsService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}

func (s *SettingsService) FetchGeneralSettings() (*models.SettingsGeneral, error) {
	return s.Repo.FetchGeneralSettings(nil)
}

func (s *SettingsService) FetchUserSettings(userID int64) (*models.SettingsUser, error) {
	return s.Repo.FetchUserSettings(nil, userID)
}

func (s *SettingsService) UpdateUserSettings(userID int64, req models.SettingsUserReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Fetch settings to confirm user is owner
	existingSettings, err := s.Repo.FetchUserSettings(nil, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find settings for given user: %w", err)
	}

	settings := models.SettingsUser{
		UserID:   userID,
		Theme:    req.Theme,
		Accent:   req.Accent,
		Timezone: req.Timezone,
		Language: req.Language,
	}

	err = s.Repo.UpdateUserSettings(tx, userID, settings)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch activity log
	changes := utils.InitChanges()

	utils.CompareChanges(existingSettings.Theme, settings.Theme, changes, "theme")
	utils.CompareChanges(utils.SafeString(existingSettings.Accent), utils.SafeString(settings.Accent), changes, "accent")
	utils.CompareChanges(existingSettings.Language, settings.Language, changes, "language")
	utils.CompareChanges(existingSettings.Timezone, settings.Timezone, changes, "timezone")

	err = s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "update",
		Category:    "user_settings",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}
