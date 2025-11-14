package tests

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"testing"
	"wealth-warden/internal/bootstrap"
	wwHttp "wealth-warden/internal/http"
	"wealth-warden/pkg/database/seeders/workers"
)

func setupTestContainer(t *testing.T) *bootstrap.Container {
	t.Helper()

	// Create container using the same bootstrap logic as production
	container, err := bootstrap.NewContainer(testCfg, testDB, testLogger)
	require.NoError(t, err, "Should create test container")

	return container
}

func setupTestServer(t *testing.T) *wwHttp.Server {
	t.Helper()

	gin.SetMode(gin.TestMode)

	container := setupTestContainer(t)
	s := wwHttp.NewServer(container, testLogger)

	return s
}

func createRootUser(t *testing.T) (userID int64, accessToken, refreshToken string) {
	t.Helper()

	ctx := context.Background()

	if err := workers.SeedRootUser(ctx, testDB, testLogger, testCfg); err != nil {
		t.Fatalf("Failed to create root user: %v", err)
	}

	var user struct {
		ID int64
	}
	err := testDB.Raw("SELECT id FROM users WHERE email = ?", testCfg.Seed.SuperAdminEmail).Scan(&user).Error
	require.NoError(t, err, "Should find seeded user")

	accessToken, refreshToken, err = testContainer.Middleware.GenerateLoginTokens(user.ID, true)
	require.NoError(t, err, "Should generate tokens")

	return user.ID, accessToken, refreshToken
}

// Clean up data between tests
func cleanupTestData(t *testing.T) {
	t.Helper()

	tables := []string{
		"transactions",
		"accounts",
		"users",
	}

	for _, table := range tables {
		testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}
