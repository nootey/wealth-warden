package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedMemberUser(ctx context.Context, db *gorm.DB, cfg *config.Config) error {

	email := cfg.Seed.MemberUserEmail
	if email == "" {
		fmt.Println("No member email provided in config, skipping ...")
		return nil
	}

	hashedPassword, err := utils.HashAndSaltPassword(cfg.Seed.MemberUserPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	// Fetch the global role ID for "member".
	var globalRoleID int64
	err = db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role member: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'member' does not exist, please seed roles first")
	}

	// Check if the member user already exists.
	var userID int64
	err = db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	// If the user doesn't exist, insert them with the global role "member"
	if userID == 0 {
		err = db.WithContext(ctx).Exec(`
			INSERT INTO users (email, password, display_name, role_id, email_confirmed, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, email, hashedPassword, "Member", globalRoleID, time.Now().UTC(), time.Now().UTC(), time.Now().UTC()).Error
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
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
