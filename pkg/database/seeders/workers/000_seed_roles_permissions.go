package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/config"

	"gorm.io/gorm"
)

func SeedRolesAndPermissions(ctx context.Context, db *gorm.DB, cfg *config.Config) error {

	type roleDef struct {
		Name string
	}

	type PermissionDef struct {
		Name        string
		Description string
		Module      string // Group permissions into modules
	}

	type rolePermSet struct {
		Role        string
		Permissions []string
	}

	globalRoles := []roleDef{
		{"super-admin"},
		{"admin"},
		{"member"},
		{"guest"},
	}

	// Define all permissions.
	permissions := []PermissionDef{

		//  Top level root permission
		{"root_access", "Root level access", "root"},

		// User management
		{"manage_users", "Manage users (global)", "users"},
		{"delete_users", "Delete users (global)", "users"},

		{"manage_roles", "Manage roles (global)", "users"},
		{"delete_roles", "Delete non default roles (global)", "users"},

		{"manage_subscriptions", "Manage subscription (global)", "users"},

		// Data modules
		// -> General data viewing
		{"view_data", "View generic app data (global)", "data"},
		{"manage_data", "Create, update and delete generic app data (global)", "data"},

		// -> Statistics
		{"view_basic_statistics", "View basic statistics (global)", "data"},
		{"view_advanced_statistics", "Gain access to advanced statistics (global)", "data"},

		// -> Logging module
		{"view_activity_logs", "View activity logs (global)", "logs"},
		{"delete_activity_logs", "Delete activity logs (global)", "logs"},

		// Exporting
		{"create_exports", "Create exports (global)", "export"},
		{"delete_exports", "Delete exports (global)", "export"},

		// Reporting
		{"create_reports", "Create reports (global)", "reporting"},
		{"delete_reports", "Delete reports (global)", "reporting"},
	}

	// Define role-to-permission mapping.
	var rolePermissionsList = []rolePermSet{
		{
			Role: "super-admin",
			Permissions: []string{
				"root_access",
			},
		},
		{
			Role: "admin",
			Permissions: []string{
				"manage_users",
				"delete_users",
				"manage_subscriptions",
				"view_data",
				"manage_data",
				"view_basic_statistics",
				"view_advanced_statistics",
				"view_activity_logs",
				"delete_activity_logs",
				"create_exports",
				"delete_exports",
				"create_reports",
				"delete_reports",
			},
		},
		{
			Role: "member",
			Permissions: []string{
				"manage_subscriptions",
				"view_data",
				"manage_data",
				"view_basic_statistics",
				"create_exports",
				"create_reports",
			},
		},
		{
			Role: "guest",
			Permissions: []string{
				"view_data",
				"view_basic_statistics",
			},
		},
	}

	// Map to store role IDs keyed by role name.
	roleIDs := make(map[string]int64)

	// Insert global roles.
	for _, role := range globalRoles {
		var roleID int64
		err := db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking role %s: %w", role.Name, err)
		}
		if roleID == 0 {
			// Insert role with is_global flag set to true.
			err = db.Exec(
				`INSERT INTO roles (name, is_default, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				role.Name, true, time.Now().UTC(), time.Now().UTC(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting role %s: %w", role.Name, err)
			}
			err = db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted role ID for %s: %w", role.Name, err)
			}
		}
		roleIDs[role.Name] = roleID
	}

	// Insert permissions and record their IDs.
	permissionIDs := make(map[string]int64)
	for _, perm := range permissions {
		var permID int64
		err := db.WithContext(ctx).Raw(`SELECT id FROM permissions WHERE name = ?`, perm.Name).Scan(&permID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking permission %s: %w", perm, err)
		}
		if permID == 0 {
			err = db.Exec(
				`INSERT INTO permissions (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				perm.Name, perm.Description, time.Now().UTC(), time.Now().UTC(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting permission %s: %w", perm, err)
			}
			err = db.WithContext(ctx).Raw(`SELECT id FROM permissions WHERE name = ?`, perm.Name).Scan(&permID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted permission ID for %s: %w", perm, err)
			}
		}
		permissionIDs[perm.Name] = permID
	}

	// Insert role-permission mappings.
	for _, rolePerm := range rolePermissionsList {
		roleID := roleIDs[rolePerm.Role]
		for _, perm := range rolePerm.Permissions {
			permID := permissionIDs[perm]
			var exists int
			err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM role_permissions WHERE role_id = ? AND permission_id = ?`, roleID, permID).Scan(&exists).Error
			if err != nil {
				return fmt.Errorf("error checking role_permission mapping for %s -> %s: %w", rolePerm.Role, perm, err)
			}
			if exists == 0 {
				err = db.Exec(
					`INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
					roleID, permID, time.Now().UTC(), time.Now().UTC(),
				).Error
				if err != nil {
					return fmt.Errorf("error inserting role_permission for %s -> %s: %w", rolePerm.Role, perm, err)
				}
			}
		}
	}

	return nil
}
