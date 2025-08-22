package workers

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedMemberUser(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {

	cfg, cfgErr := config.LoadConfig(nil)
	if cfgErr != nil {
		return cfgErr
	}

	email := cfg.Seed.MemberUserEmail
	hashedPassword, err := utils.HashAndSaltPassword(cfg.Seed.MemberUserPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	// Fetch the global role ID for "member".
	var globalRoleID int64
	err = db.Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role member: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'member' does not exist, please seed roles first")
	}

	// Check if the member user already exists.
	var userID int64
	err = db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	// If the user doesn't exist, insert them with the global role "member"
	if userID == 0 {
		err = db.Exec(`
			INSERT INTO users (username, email, password, display_name, role_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, "Member", email, hashedPassword, "Member", globalRoleID, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
	}

	return nil
}
