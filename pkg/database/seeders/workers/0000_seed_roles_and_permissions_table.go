package workers

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func SeedRolesAndPermissions(ctx context.Context, db *gorm.DB) error {
	// Define a helper struct for role definitions.
	type roleDef struct {
		Name     string
		IsGlobal bool
	}

	// Global roles – related to user operations.
	globalRoles := []roleDef{
		{"super-admin", true},
		{"admin", true},
		{"member", true},
		{"guest", true},
	}

	// Organization roles – related to organization-specific operations.
	orgRoles := []roleDef{
		{"owner", false},
		{"editor", false},
		{"viewer", false},
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
				`INSERT INTO roles (name, is_global, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				role.Name, true, time.Now(), time.Now(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting role %s: %w", role.Name, err)
			}
			err = db.Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted role ID for %s: %w", role.Name, err)
			}
		}
		roleIDs[role.Name] = roleID
	}

	// Insert organization roles.
	for _, role := range orgRoles {
		var roleID uint
		err := db.Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking role %s: %w", role.Name, err)
		}
		if roleID == 0 {
			// Insert role with is_global flag set to false.
			err = db.Exec(
				`INSERT INTO roles (name, is_global, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				role.Name, false, time.Now(), time.Now(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting role %s: %w", role.Name, err)
			}
			err = db.Raw(`SELECT id FROM roles WHERE name = ?`, role.Name).Scan(&roleID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted role ID for %s: %w", role.Name, err)
			}
		}
		roleIDs[role.Name] = roleID
	}

	// Define all permissions only once.
	permissions := map[string]string{
		// Global permissions
		"root_access":         "Super admin root level access",
		"manage_users":        "Manage users (global)",
		"manage_roles":        "Manage roles (global)",
		"manage_subscription": "Manage subscription (global)",
		"view_data":           "View data (global)",
		"view_audit_logs":     "View audit logs (global)",
		"view_access_logs":    "View access logs (global)",
		"create_exports":      "Create exports (global)",
		"delete_exports":      "Delete exports (global)",
		"create_reports":      "Create reports (global)",
		"delete_reports":      "Delete reports (global)",
		// Organization permissions
		"manage_self_organization":     "Manage own organization",
		"invite_organization_members":  "Invite members to organization",
		"remove_organization_members":  "Remove members from organization",
		"view_organization_finances":   "View organization finances",
		"edit_organization_finances":   "Edit organization finances",
		"delete_organization_finances": "Delete organization finances",
	}

	// Insert permissions and record their IDs.
	permissionIDs := make(map[string]uint)
	for perm, desc := range permissions {
		var permID uint
		err := db.Raw(`SELECT id FROM permissions WHERE name = ?`, perm).Scan(&permID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking permission %s: %w", perm, err)
		}
		if permID == 0 {
			err = db.Exec(
				`INSERT INTO permissions (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)`,
				perm, desc, time.Now(), time.Now(),
			).Error
			if err != nil {
				return fmt.Errorf("error inserting permission %s: %w", perm, err)
			}
			err = db.Raw(`SELECT id FROM permissions WHERE name = ?`, perm).Scan(&permID).Error
			if err != nil {
				return fmt.Errorf("error retrieving inserted permission ID for %s: %w", perm, err)
			}
		}
		permissionIDs[perm] = permID
	}

	// Define role-to-permission mapping.
	// Each permission is referenced by its key defined above.
	rolePermissions := map[string][]string{
		"super-admin": {"root_access"},
		"admin":       {"manage_users", "manage_roles", "manage_subscription", "view_data", "view_audit_logs", "view_access_logs", "create_exports", "delete_exports", "create_reports", "delete_reports"},
		"member":      {"manage_subscription", "view_data", "view_audit_logs", "create_exports", "delete_exports", "create_reports", "delete_reports"},
		"guest":       {"view_data"},
		"owner":       {"manage_self_organization", "invite_organization_members", "remove_organization_members", "view_organization_finances", "edit_organization_finances", "delete_organization_finances"},
		"editor":      {"view_organization_finances", "edit_organization_finances"},
		"viewer":      {"view_organization_finances"},
	}

	// Insert role-permission mappings.
	for role, perms := range rolePermissions {
		roleID := roleIDs[role]
		for _, perm := range perms {
			permID := permissionIDs[perm]
			var exists int
			err := db.Raw(`SELECT COUNT(*) FROM role_permissions WHERE role_id = ? AND permission_id = ?`, roleID, permID).Scan(&exists).Error
			if err != nil {
				return fmt.Errorf("error checking role_permission mapping for %s -> %s: %w", role, perm, err)
			}
			if exists == 0 {
				err = db.Exec(
					`INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
					roleID, permID, time.Now(), time.Now(),
				).Error
				if err != nil {
					return fmt.Errorf("error inserting role_permission for %s -> %s: %w", role, perm, err)
				}
			}
		}
	}

	return nil
}
