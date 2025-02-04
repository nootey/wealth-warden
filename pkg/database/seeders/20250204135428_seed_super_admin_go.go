package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pressly/goose/v3"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

func init() {
	goose.AddMigrationContext(upSeedSuperAdminGo, downSeedSuperAdminGo)
}

func upSeedSuperAdminGo(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	cfg := config.LoadConfig()

	// Hash the password using your utility function
	hashedPassword, err := utils.HashAndSaltPassword(cfg.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the super admin user into the database
	_, err = tx.Exec(`
        INSERT INTO users (username, email, password, role)
        VALUES (?, ?, ?, ?)
    `, "Support", "support@wealth-warden.com", hashedPassword, "super-admin")

	if err != nil {
		return fmt.Errorf("failed to insert super admin: %w", err)
	}

	return nil
}

func downSeedSuperAdminGo(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec(`DELETE FROM users WHERE email = ?`, "admin@example.com")
	if err != nil {
		return fmt.Errorf("failed to delete super admin: %w", err)
	}
	return nil
}
