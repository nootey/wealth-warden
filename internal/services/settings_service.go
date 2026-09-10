package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	_ "time/tzdata"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/config"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SettingsServiceInterface interface {
	FetchGeneralSettings(ctx context.Context) (*models.SettingsGeneral, error)
	FetchUserSettings(ctx context.Context, userID int64) (*models.SettingsUser, error)
	FetchAvailableTimezones(ctx context.Context) ([]models.TimezoneInfo, error)
	FetchAvailableCurrencies(ctx context.Context) ([]models.CurrencyInfo, error)
	UpdatePreferenceSettings(ctx context.Context, userID int64, req models.PreferenceSettingsReq) error
	UpdateProfileSettings(ctx context.Context, userID int64, req models.ProfileSettingsReq) error
}

type SettingsService struct {
	cfg           *config.Config
	logger        *zap.Logger
	repo          repositories.SettingsRepositoryInterface
	userRepo      repositories.UserRepositoryInterface
	jobDispatcher jobqueue.Dispatcher
	sessionStore  *sessions.Store
}

func NewSettingsService(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repositories.SettingsRepository,
	userRepo *repositories.UserRepository,
	jobDispatcher jobqueue.Dispatcher,
	sessionStore *sessions.Store,
) *SettingsService {
	return &SettingsService{
		cfg:           cfg,
		logger:        logger,
		repo:          repo,
		userRepo:      userRepo,
		jobDispatcher: jobDispatcher,
		sessionStore:  sessionStore,
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
			fmt.Printf("settings_service: Loading timezone %s failed: %v", tzName, err)
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

func (s *SettingsService) FetchAvailableCurrencies(ctx context.Context) ([]models.CurrencyInfo, error) {
	currencies := utils.GetIsoCurrencies()

	result := make([]models.CurrencyInfo, 0, len(currencies))
	for code, name := range currencies {
		result = append(result, models.CurrencyInfo{
			Value: code,
			Label: fmt.Sprintf("%s - %s", code, name),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Value < result[j].Value
	})

	return result, nil
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.Wrap(apperr.NotFound, "Settings not found", err)
		}
		return err
	}

	if req.DefaultCurrency != "" && req.DefaultCurrency != existingSettings.DefaultCurrency {
		tx.Rollback()
		return apperr.New(apperr.Invalid, "default currency currently cannot be changed after initial setup")
	}

	settings := models.SettingsUser{
		UserID:                userID,
		Theme:                 req.Theme,
		Accent:                req.Accent,
		Timezone:              req.Timezone,
		Language:              req.Language,
		DefaultCurrency:       existingSettings.DefaultCurrency,
		DefaultSheetSeparator: req.DefaultSheetSeparator,
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
	utils.CompareChanges(existingSettings.DefaultCurrency, settings.DefaultCurrency, changes, "default_currency")
	utils.CompareChanges(existingSettings.DefaultSheetSeparator, settings.DefaultSheetSeparator, changes, "default_sheet_separator")

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "update",
		Category:    "user_settings",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	if req.Timezone != "" && req.Timezone != existingSettings.Timezone {
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.RecalculateTemplateTimezoneArgs{
			UserID:      userID,
			OldTimezone: existingSettings.Timezone,
			NewTimezone: req.Timezone,
		})
		if err != nil {
			s.logger.Warn("Failed to dispatch template timezone recalculation job", zap.Error(err))
		}
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.Wrap(apperr.NotFound, "User not found", err)
		}
		return err
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

	if req.Password != nil && *req.Password != "" {
		if req.PasswordConfirmation == nil || *req.Password != *req.PasswordConfirmation {
			tx.Rollback()
			return apperr.New(apperr.Validation, "password confirmation must match provided password")
		}
		hashed, err := utils.HashAndSaltPassword(*req.Password)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to hash password: %w", err)
		}
		if err := s.userRepo.UpdateUserPassword(ctx, tx, existingUser.ID, hashed); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if req.Password != nil && *req.Password != "" {
		if err := s.sessionStore.DeleteAllForUser(ctx, existingUser.ID); err != nil {
			return err
		}
	}

	// Dispatch activity log
	changes := utils.InitChanges()

	utils.CompareChanges(existingUser.Email, u.Email, changes, "email")
	utils.CompareChanges(existingUser.DisplayName, u.DisplayName, changes, "display_name")

	var description *string
	if req.Password != nil && *req.Password != "" {
		d := "Password changed"
		description = &d
	}

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "update",
		Category:    "user",
		Description: description,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}
