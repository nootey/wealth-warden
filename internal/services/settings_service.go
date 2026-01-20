package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"time"
	_ "time/tzdata"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/version"

	"go.uber.org/zap"
)

type SettingsServiceInterface interface {
	FetchGeneralSettings(ctx context.Context) (*models.SettingsGeneral, error)
	FetchUserSettings(ctx context.Context, userID int64) (*models.SettingsUser, error)
	FetchAvailableTimezones(ctx context.Context) ([]models.TimezoneInfo, error)
	UpdatePreferenceSettings(ctx context.Context, userID int64, req models.PreferenceSettingsReq) error
	UpdateProfileSettings(ctx context.Context, userID int64, req models.ProfileSettingsReq) error
	RestoreDatabaseBackup(ctx context.Context, userID int64, backupName string) error
}

type SettingsService struct {
	cfg           *config.Config
	logger        *zap.Logger
	repo          repositories.SettingsRepositoryInterface
	userRepo      repositories.UserRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
}

func NewSettingsService(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repositories.SettingsRepository,
	userRepo *repositories.UserRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
) *SettingsService {
	return &SettingsService{
		cfg:           cfg,
		logger:        logger,
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

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
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

func (s *SettingsService) GetDatabaseBackups(ctx context.Context) ([]models.BackupInfo, error) {
	backupsPath := filepath.Join("storage", "backups")

	// Check if backups directory exists
	if _, err := os.Stat(backupsPath); os.IsNotExist(err) {
		// Return empty list if directory doesn't exist yet
		return []models.BackupInfo{}, nil
	}

	// Read all entries in backups directory
	entries, err := os.ReadDir(backupsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	backups := []models.BackupInfo{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files, only process directories
		}

		backupName := entry.Name()
		metadataPath := filepath.Join(backupsPath, backupName, "metadata.json")

		// Read metadata file
		metadataBytes, err := os.ReadFile(metadataPath)
		if err != nil {
			s.logger.Warn("Failed to read metadata for backup",
				zap.String("backup_name", backupName),
				zap.Error(err),
			)
			continue // Skip this backup if metadata can't be read
		}

		var metadata models.BackupMetadata
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			s.logger.Warn("Failed to parse metadata for backup",
				zap.String("backup_name", backupName),
				zap.Error(err),
			)
			continue // Skip this backup if metadata can't be parsed
		}

		backups = append(backups, models.BackupInfo{
			Name:     backupName,
			Metadata: metadata,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Metadata.CreatedAt.After(backups[j].Metadata.CreatedAt)
	})

	return backups, nil
}

func (s *SettingsService) CreateDatabaseBackup(ctx context.Context, userID int64) error {

	// Get versions
	appVersion := version.Version
	commitSHA := version.CommitSHA
	buildTime := version.BuildTime

	dbVersion, err := s.repo.FetchGooseVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch database version: %w", err)
	}

	// Create backup directory structure
	timestamp := time.Now().Format("2006-01-02_150405")
	shortCommit := commitSHA
	if len(shortCommit) > 7 {
		shortCommit = shortCommit[:7]
	}
	backupName := fmt.Sprintf("backup_%s_%s", timestamp, shortCommit)
	backupPath := filepath.Join("storage", "backups", backupName)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	dumpPath := filepath.Join(backupPath, "dump.sql")
	metadataPath := filepath.Join(backupPath, "metadata.json")

	dbHost := s.cfg.Postgres.Host
	dbPort := s.cfg.Postgres.Port
	dbName := s.cfg.Postgres.Database
	dbUser := s.cfg.Postgres.User

	cmd := exec.CommandContext(ctx, "pg_dump",
		"-h", dbHost,
		"-p", strconv.Itoa(dbPort),
		"-U", dbUser,
		"-d", dbName,
		"-f", dumpPath,
		"--clean",
		"--if-exists",
	)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.cfg.Postgres.Password))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create database dump: %w, output: %s", err, string(output))
	}

	fileInfo, err := os.Stat(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to stat dump file: %w", err)
	}

	metadata := models.BackupMetadata{
		AppVersion: appVersion,
		CommitSHA:  commitSHA,
		BuildTime:  buildTime,
		DBVersion:  dbVersion,
		CreatedAt:  time.Now(),
		BackupSize: fileInfo.Size(),
	}

	// Write metadata to JSON
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	s.logger.Info("Backup created successfully",
		zap.String("backup_path", backupPath),
		zap.String("app_version", appVersion),
		zap.String("commit_sha", commitSHA),
		zap.String("build_time", buildTime),
		zap.Int64("db_version", dbVersion),
		zap.Int64("backup_size_bytes", fileInfo.Size()),
	)

	return nil
}

func (s *SettingsService) RestoreDatabaseBackup(ctx context.Context, userID int64, backupName string) error {
	// Construct backup paths
	backupPath := filepath.Join("storage", "backups", backupName)
	dumpPath := filepath.Join(backupPath, "dump.sql")
	metadataPath := filepath.Join(backupPath, "metadata.json")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory does not exist: %s", backupName)
	}

	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("dump file does not exist in backup: %s", backupName)
	}

	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata models.BackupMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	currentDBVersion, err := s.repo.FetchGooseVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch current database version: %w", err)
	}

	// Verify database version matches
	if metadata.DBVersion != currentDBVersion {
		return fmt.Errorf("database version mismatch: backup version %d, current version %d",
			metadata.DBVersion, currentDBVersion)
	}

	// Perform the restore
	dbHost := s.cfg.Postgres.Host
	dbPort := s.cfg.Postgres.Port
	dbName := s.cfg.Postgres.Database
	dbUser := s.cfg.Postgres.User

	cmd := exec.CommandContext(ctx, "psql",
		"-h", dbHost,
		"-p", strconv.Itoa(dbPort),
		"-U", dbUser,
		"-d", dbName,
		"-f", dumpPath,
	)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.cfg.Postgres.Password))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restore database: %w, output: %s", err, string(output))
	}

	s.logger.Info("Database restored successfully",
		zap.String("backup_dir", backupName),
		zap.String("backup_app_version", metadata.AppVersion),
		zap.String("backup_commit_sha", metadata.CommitSHA),
		zap.Int64("db_version", metadata.DBVersion),
		zap.Time("backup_created_at", metadata.CreatedAt),
	)

	// Schedule application restart after database restore
	go func() {
		time.Sleep(3 * time.Second)
		s.logger.Warn("Initiating application restart after database restore")
		os.Exit(0)
	}()

	return nil
}
