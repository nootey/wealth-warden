package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedRootUser(ctx context.Context, db *gorm.DB, cfg *config.Config) error {

	email := cfg.Seed.SuperAdminEmail

	// Hash the Super Admin password.
	hashedPassword, err := utils.HashAndSaltPassword(cfg.Seed.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Fetch the global role ID for "super-admin".
	var globalRoleID int64
	err = db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, "super-admin").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role super-admin: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'super-admin' does not exist, please seed roles first")
	}

	// Check if the Super Admin user already exists.
	var userID int64
	err = db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing super admin user: %w", err)
	}

	// If the user doesn't exist, insert them with the global role "super-admin"
	if userID == 0 {
		err = db.WithContext(ctx).Exec(`
			INSERT INTO users (email, password, display_name, role_id, email_confirmed, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, email, hashedPassword, "Support", globalRoleID, time.Now().UTC(), time.Now().UTC(), time.Now().UTC()).Error
		if err != nil {
			return fmt.Errorf("failed to insert super admin user: %w", err)
		}

		err = db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
		if err != nil {
			return fmt.Errorf("failed to refetch super admin id after insert: %w", err)
		}
	}

	// Seed default user settings if not present
	err = db.WithContext(ctx).Exec(`
		INSERT INTO settings_user (user_id, theme, accent, language, timezone, created_at, updated_at)
		VALUES (?, 'system', NULL, 'en', 'UTC', ?, ?)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, time.Now().UTC(), time.Now().UTC()).Error
	if err != nil {
		return fmt.Errorf("failed to insert root user default settings: %w", err)
	}

	return nil
}
