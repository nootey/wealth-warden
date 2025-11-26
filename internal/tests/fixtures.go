package tests

import (
	"context"
	"testing"
	"wealth-warden/pkg/database/seeders/workers"

	"github.com/stretchr/testify/require"
)

func SetupRootUser() error {
	ctx := context.Background()

	if err := workers.SeedRootUser(ctx, DB, Logger, Config); err != nil {
		return err
	}

	return nil
}

func CreateRootUser(t *testing.T) (userID int64, accessToken, refreshToken string) {
	t.Helper()

	var user struct {
		ID     int64
		Email  string
		RoleID int64
	}
	err := DB.Raw("SELECT id, email, role_id FROM users WHERE email = ?", Config.Seed.SuperAdminEmail).Scan(&user).Error
	require.NoError(t, err, "Should find seeded user")

	var role struct {
		ID   int64
		Name string
	}
	err = DB.Raw("SELECT id, name FROM roles WHERE id = ?", user.RoleID).Scan(&role).Error
	require.NoError(t, err, "Should find user's role")

	var permCount int64
	DB.Raw("SELECT COUNT(*) FROM role_permissions WHERE role_id = ?", user.RoleID).Scan(&permCount)

	var permNames []string
	DB.Raw(`
		SELECT p.name 
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?
	`, user.RoleID).Scan(&permNames)

	if Container == nil {
		t.Fatal("Container is nil!")
	}
	if Container.Middleware == nil {
		t.Fatal("Container.Middleware is nil!")
	}

	accessToken, refreshToken, err = Container.Middleware.GenerateLoginTokens(user.ID, true)
	require.NoError(t, err, "Should generate tokens")

	return user.ID, accessToken, refreshToken
}
