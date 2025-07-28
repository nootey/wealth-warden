package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedMemberUser(ctx context.Context, db *gorm.DB) error {

	creds, err := LoadSeederCredentials()
	if err != nil {
		return fmt.Errorf("failed to load seeder credentials: %w", err)
	}
	email, ok := creds["MEMBER_EMAIL"]
	if !ok || email == "" {
		return fmt.Errorf("MEMBER_EMAIL not set in seeder credentials")
	}
	password, ok := creds["MEMBER_PASSWORD"]
	if !ok || password == "" {
		return fmt.Errorf("MEMBER_PASSWORD not set in seeder credentials")
	}

	// Hash the Member password.
	hashedPassword, err := utils.HashAndSaltPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Fetch the global role ID for "member".
	var globalRoleID uint
	err = db.Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role member: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'member' does not exist, please seed roles first")
	}

	// Check if the member user already exists.
	var userID uint
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
