package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedSuperAdmin(ctx context.Context, db *gorm.DB) error {

	creds, err := LoadSeederCredentials()
	if err != nil {
		return fmt.Errorf("failed to load seeder credentials: %w", err)
	}
	email, ok := creds["SUPER_ADMIN_EMAIL"]
	if !ok || email == "" {
		return fmt.Errorf("SUPER_ADMIN_EMAIL not set in seeder credentials")
	}
	password, ok := creds["SUPER_ADMIN_PASSWORD"]
	if !ok || password == "" {
		return fmt.Errorf("SUPER_ADMIN_PASSWORD not set in seeder credentials")
	}

	// Hash the Super Admin password.
	hashedPassword, err := utils.HashAndSaltPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Fetch the global role ID for "super-admin".
	var globalRoleID uint
	err = db.Raw(`SELECT id FROM roles WHERE name = ? AND is_global = true`, "super-admin").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role super-admin: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'super-admin' does not exist, please seed roles first")
	}

	// Step 1: Check if the "Super Admin" organization already exists.
	var organizationID uint
	err = db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Super Admin").Scan(&organizationID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing Super Admin organization: %w", err)
	}

	// Step 2: If the organization doesn't exist, create it.
	if organizationID == 0 {
		err = db.Exec(`
			INSERT INTO organizations (name, organization_type, created_at)
			VALUES (?, ?, ?)
		`, "Super Admin", "solo", time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert Super Admin organization: %w", err)
		}

		// Retrieve the newly inserted organization ID.
		err = db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Super Admin").Scan(&organizationID).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve new Super Admin organization ID: %w", err)
		}
	}

	// Step 3: Check if the Super Admin user already exists.
	var userID uint
	err = db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing super admin user: %w", err)
	}

	// Step 4: If the user doesn't exist, insert them with the global role "super-admin"
	// and assign the primary_organization_id.
	if userID == 0 {
		err = db.Exec(`
			INSERT INTO users (username, email, password, display_name, role_id, primary_organization_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "Support", email, hashedPassword, "Super Admin", globalRoleID, organizationID, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert super admin user: %w", err)
		}

		// Retrieve the newly inserted user ID.
		err = db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve new super admin user ID: %w", err)
		}
	}

	// Fetch the organization role ID for "owner" (non-global role).
	var orgRoleID uint
	err = db.Raw(`SELECT id FROM roles WHERE name = ? AND is_global = false`, "owner").Scan(&orgRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch organization role owner: %w", err)
	}
	if orgRoleID == 0 {
		return fmt.Errorf("organization role 'owner' does not exist, please seed roles first")
	}

	// Step 5: Check if the user is already assigned to the Super Admin organization with the "owner" role.
	var exists int
	err = db.Raw(`
		SELECT COUNT(*) FROM organization_users 
		WHERE user_id = ? AND organization_id = ? AND role_id = ?
	`, userID, organizationID, orgRoleID).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check existing organization membership: %w", err)
	}

	// Step 6: If not already assigned, insert the user into the organization with the "owner" role.
	if exists == 0 {
		err = db.Exec(`
			INSERT INTO organization_users (user_id, organization_id, role_id, created_at)
			VALUES (?, ?, ?, ?)
		`, userID, organizationID, orgRoleID, time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to assign organization role (owner) to user: %w", err)
		}
	}

	// Step 7: Check if the `user_secrets` entry exists for this user.
	var secretExists int
	err = db.Raw(`SELECT COUNT(*) FROM user_secrets WHERE user_id = ?`, userID).Scan(&secretExists).Error
	if err != nil {
		return fmt.Errorf("failed to check existing user secrets for Super Admin: %w", err)
	}

	// Step 8: If the `user_secrets` entry doesn't exist, create it.
	if secretExists == 0 {
		err = db.Exec(`
			INSERT INTO user_secrets (user_id, budget_initialized, allow_log, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, userID, false, true, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert user secrets for Super Admin: %w", err)
		}
	}

	return nil
}
