package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

func SeedSuperAdmin(ctx context.Context, db *gorm.DB) error {
	cfg := config.LoadConfig()

	hashedPassword, err := utils.HashAndSaltPassword(cfg.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err = db.Exec(`
        INSERT INTO users (username, email, password, role)
        VALUES (?, ?, ?, ?)
    `, "Support", "support@wealth-warden.com", hashedPassword, "super-admin").Error

	if err != nil {
		return fmt.Errorf("failed to insert super admin: %w", err)
	}

	return nil
}
