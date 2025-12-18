package services

import (
	"context"
	"fmt"
	"sort"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type SettingsServiceInterface interface {
	FetchGeneralSettings(ctx context.Context) (*models.SettingsGeneral, error)
	FetchUserSettings(ctx context.Context, userID int64) (*models.SettingsUser, error)
	FetchAvailableTimezones(ctx context.Context) ([]models.TimezoneInfo, error)
	UpdatePreferenceSettings(ctx context.Context, userID int64, req models.PreferenceSettingsReq) error
	UpdateProfileSettings(ctx context.Context, userID int64, req models.ProfileSettingsReq) error
}

type SettingsService struct {
	repo          repositories.SettingsRepositoryInterface
	userRepo      repositories.UserRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobs.JobDispatcher
}

func NewSettingsService(
	repo *repositories.SettingsRepository,
	userRepo *repositories.UserRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobs.JobDispatcher,
) *SettingsService {
	return &SettingsService{
		repo:          repo,
		userRepo:      userRepo,
		loggingRepo:   loggingRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ SettingsServiceInterface = (*SettingsService)(nil)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (s *SettingsService) FetchGeneralSettings(ctx context.Context) (*models.SettingsGeneral, error) {
	return s.repo.FetchGeneralSettings(ctx, nil)
}

func (s *SettingsService) FetchUserSettings(ctx context.Context, userID int64) (*models.SettingsUser, error) {
	return s.repo.FetchUserSettings(ctx, nil, userID)
}

func (s *SettingsService) FetchAvailableTimezones(ctx context.Context) ([]models.TimezoneInfo, error) {

	var timezones []models.TimezoneInfo

	// Get all IANA timezone identifiers
	tzNames := utils.GetIANATimezones()

	now := time.Now()

	for _, tzName := range tzNames {
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			fmt.Println(fmt.Sprintf("settings_service: Loading timezone %s failed: %v", tzName, err))
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

func (s *SettingsService) UpdatePreferenceSettings(ctx context.Context, userID int64, req models.PreferenceSettingsReq) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Fetch settings to confirm user is owner
	existingSettings, err := s.repo.FetchUserSettings(ctx, nil, userID)
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

	err = s.repo.UpdateUserSettings(ctx, tx, userID, settings)
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

	err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *SettingsService) UpdateProfileSettings(ctx context.Context, userID int64, req models.ProfileSettingsReq) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Fetch settings to confirm user is owner
	existingUser, err := s.userRepo.FindUserByID(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing user: %w", err)
	}

	u := models.User{
		ID:          existingUser.ID,
		DisplayName: req.DisplayName,
		RoleID:      existingUser.RoleID,
	}

	if req.EmailUpdated {
		u.Email = req.Email
	}

	_, err = s.userRepo.UpdateUser(ctx, tx, u)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Dispatch activity log
	changes := utils.InitChanges()

	utils.CompareChanges(existingUser.Email, u.Email, changes, "email")
	utils.CompareChanges(existingUser.DisplayName, u.DisplayName, changes, "display_name")

	err = s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "update",
		Category:    "user",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}
