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
	goose.AddMigrationContext(upSeedMember, downSeedMember)
}

func upSeedMember(ctx context.Context, tx *sql.Tx) error {
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
    `, "Member", "member@wealth-warden.com", hashedPassword, "member")

	if err != nil {
		return fmt.Errorf("failed to insert member: %w", err)
	}

	return nil

}

func downSeedMember(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.

	_, err := tx.Exec(`DELETE FROM users WHERE email = ?`, "member@wealth-warden.com")
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	return nil
}
