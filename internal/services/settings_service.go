package services

import (
	"fmt"
	"sort"
	"time"
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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (s *SettingsService) FetchAvailableTimezones() ([]models.TimezoneInfo, error) {

	var timezones []models.TimezoneInfo

	// Get all IANA timezone identifiers
	tzNames := utils.GetIANATimezones()

	now := time.Now()

	for _, tzName := range tzNames {
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			continue
		}

		// Get current offset
		_, offset := now.In(loc).Zone()
		offsetHours := offset / 3600
		offsetMinutes := (offset % 3600) / 60

		// Format offset as +05:30 or -08:00
		offsetStr := fmt.Sprintf("%+03d:%02d", offsetHours, abs(offsetMinutes))

		timezones = append(timezones, models.TimezoneInfo{
			Value:       tzName,
			Label:       fmt.Sprintf("(UTC%s) %s", offsetStr, tzName),
			Offset:      offset,
			DisplayName: tzName,
		})
	}

	// Sort by offset, then by name
	sort.Slice(timezones, func(i, j int) bool {
		if timezones[i].Offset != timezones[j].Offset {
			return timezones[i].Offset < timezones[j].Offset
		}
		return timezones[i].Value < timezones[j].Value
	})

	return timezones, nil
}
