package workers

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func SeedRolesAndPermissions(ctx context.Context, db *gorm.DB) error {

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
		{"manage_roles", "Manage roles (global)", "users"},
		{"manage_subscriptions", "Manage subscription (global)", "users"},

		// Data modules
		// -> General data viewing
		{"view_data", "View generic app data (global)", "data"},

		// -> Logging module
		{"view_audit_logs", "View audit logs (global)", "logs"},
		{"delete_audit_logs", "Delete audit logs (global)", "logs"},
		{"view_access_logs", "View access logs (global)", "logs"},
		{"delete_access_logs", "Delete access logs (global)", "logs"},

		// -> Heart rate module
		{"view_heart_rate_data", "View heart rate data", "heart_rate"},
		{"analyze_heart_rate_data", "Access HR analytics", "heart_rate"},

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
				"manage_roles",
				"manage_subscriptions",
				"view_data",
				"view_audit_logs",
				"delete_audit_logs",
				"view_access_logs",
				"delete_access_logs",
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
				"view_audit_logs",
				"create_exports",
				"create_reports",
			},
		},
		{
			Role: "guest",
			Permissions: []string{
				"view_data",
			},
		},
	}

	// Map to store role IDs keyed by role name.
	roleIDs := make(map[string]uint)

	// Insert global roles.
	for _, role := range globalRoles {
		var roleID uint
		err := db.Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking role %s: %w", role.Name, err)
		}
		if roleID == 0 {
			// Insert role with is_global flag set to true.
			err = db.Exec(
				`INSERT INTO roles (name, created_at, updated_at) VALUES (?, ?, ?)`,
				role.Name, time.Now(), time.Now(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting role %s: %w", role.Name, err)
			}
			fmt.Printf("Inserted role: %s\n", role.Name)
			err = db.Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted role ID for %s: %w", role.Name, err)
			}
		}
		roleIDs[role.Name] = roleID
	}

	// Insert permissions and record their IDs.
	permissionIDs := make(map[string]uint)
	for _, perm := range permissions {
		var permID uint
		err := db.Raw(`SELECT id FROM permissions WHERE name = ?`, perm.Name).Scan(&permID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking permission %s: %w", perm, err)
		}
		if permID == 0 {
			err = db.Exec(
				`INSERT INTO permissions (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				perm.Name, perm.Description, time.Now(), time.Now(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting permission %s: %w", perm, err)
			}
			fmt.Printf("Inserted permission: %s\n", perm.Name)
			err = db.Raw(`SELECT id FROM permissions WHERE name = ?`, perm.Name).Scan(&permID).Error
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
			err := db.Raw(`SELECT COUNT(*) FROM role_permissions WHERE role_id = ? AND permission_id = ?`, roleID, permID).Scan(&exists).Error
			if err != nil {
				return fmt.Errorf("error checking role_permission mapping for %s -> %s: %w", rolePerm.Role, perm, err)
			}
			if exists == 0 {
				err = db.Exec(
					`INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
					roleID, permID, time.Now(), time.Now(),
				).Error
				if err != nil {
					return fmt.Errorf("error inserting role_permission for %s -> %s: %w", rolePerm.Role, perm, err)
				}
				fmt.Printf("Assigned permission: %s to role: %s\n", perm, rolePerm.Role)
			}
		}
	}

	return nil
}
