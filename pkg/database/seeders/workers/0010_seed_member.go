package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

func SeedMember(ctx context.Context, db *gorm.DB) error {
	// This code is executed when the migration is applied.

	cfg := config.LoadConfig()

	// Hash the password using your utility function
	hashedPassword, err := utils.HashAndSaltPassword(cfg.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the super admin user into the database
	err = db.Exec(`
        INSERT INTO users (username, email, password, role)
        VALUES (?, ?, ?, ?)
    `, "Member", "member@wealth-warden.com", hashedPassword, "member").Error

	if err != nil {
		return fmt.Errorf("failed to insert member: %w", err)
	}

	return nil

}
