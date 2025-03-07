package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedMember(ctx context.Context, db *gorm.DB) error {
	cfg := config.LoadConfig()

	// Hash the Member password.
	hashedPassword, err := utils.HashAndSaltPassword(cfg.SuperAdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Fetch the global role ID for "member".
	var globalRoleID uint
	err = db.Raw(`SELECT id FROM roles WHERE name = ? AND is_global = true`, "member").Scan(&globalRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch global role member: %w", err)
	}
	if globalRoleID == 0 {
		return fmt.Errorf("global role 'member' does not exist, please seed roles first")
	}

	// Step 1: Check if the "Member" organization exists.
	var organizationID uint
	err = db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Member").Scan(&organizationID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing Member organization: %w", err)
	}

	// Step 2: If the organization doesn't exist, create it.
	if organizationID == 0 {
		err = db.Exec(`
			INSERT INTO organizations (name, organization_type, created_at)
			VALUES (?, ?, ?)
		`, "Member", "solo", time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert Member organization: %w", err)
		}

		// Retrieve the newly inserted organization ID.
		err = db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Member").Scan(&organizationID).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve new Member organization ID: %w", err)
		}
	}

	// Step 3: Check if the member user already exists.
	var userID uint
	err = db.Raw(`SELECT id FROM users WHERE email = ?`, "member@wealth-warden.com").Scan(&userID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	// Step 4: If the user doesn't exist, insert them with the global role "member" and the primary_organization_id.
	if userID == 0 {
		err = db.Exec(`
			INSERT INTO users (username, email, password, display_name, role_id, primary_organization_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "Member", "member@wealth-warden.com", hashedPassword, "Member", globalRoleID, organizationID, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		// Retrieve the newly inserted user ID.
		err = db.Raw(`SELECT id FROM users WHERE email = ?`, "member@wealth-warden.com").Scan(&userID).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve new member user ID: %w", err)
		}
	}

	// Step 5: Fetch the organization role ID for "owner" (non-global role).
	var orgRoleID uint
	err = db.Raw(`SELECT id FROM roles WHERE name = ? AND is_global = false`, "owner").Scan(&orgRoleID).Error
	if err != nil {
		return fmt.Errorf("failed to fetch organization role owner: %w", err)
	}
	if orgRoleID == 0 {
		return fmt.Errorf("organization role 'owner' does not exist, please seed roles first")
	}

	// Step 6: Check if the user is already assigned to the Member organization with the "owner" role.
	var exists int
	err = db.Raw(`
		SELECT COUNT(*) FROM organization_users 
		WHERE user_id = ? AND organization_id = ? AND role_id = ?
	`, userID, organizationID, orgRoleID).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check existing organization membership: %w", err)
	}

	// Step 7: If not already assigned, insert the user into the organization with the "owner" role.
	if exists == 0 {
		err = db.Exec(`
			INSERT INTO organization_users (user_id, organization_id, role_id, created_at)
			VALUES (?, ?, ?, ?)
		`, userID, organizationID, orgRoleID, time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to assign organization role (owner) to user: %w", err)
		}
	}

	// Step 8: Check if the user_secrets entry exists for this user.
	var secretExists int
	err = db.Raw(`SELECT COUNT(*) FROM user_secrets WHERE user_id = ?`, userID).Scan(&secretExists).Error
	if err != nil {
		return fmt.Errorf("failed to check existing user secrets for member: %w", err)
	}

	// Step 9: If the user_secrets entry doesn't exist, create it.
	if secretExists == 0 {
		err = db.Exec(`
			INSERT INTO user_secrets (user_id, budget_initialized, allow_log, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, userID, false, true, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to insert user secrets for Member: %w", err)
		}
	}

	return nil
}
