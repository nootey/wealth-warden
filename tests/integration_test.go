package tests

import (
	"testing"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	if err := tests.Setup(); err != nil {
		panic(err)
	}
}

func TestDatabaseConnection(t *testing.T) {
	require.NotNil(t, tests.DB, "Test database should be initialized")

	sqlDB, err := tests.DB.DB()
	require.NoError(t, err)

	err = sqlDB.Ping()
	assert.NoError(t, err, "Should be able to ping test database")

	t.Log("Database connection works!")
}

func TestMigrationsRan(t *testing.T) {
	var exists bool
	err := tests.DB.Raw(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )`).Scan(&exists).Error

	require.NoError(t, err)
	assert.True(t, exists, "Users table should exist after migrations")

	t.Log("Migrations ran successfully!")
}

func TestCreateRootUser(t *testing.T) {
	tests.CleanupData(t)

	userID, accessToken, refreshToken := tests.CreateRootUser(t)

	assert.NotZero(t, userID, "Should have user ID")
	assert.NotEmpty(t, accessToken, "Should have access token")
	assert.NotEmpty(t, refreshToken, "Should have refresh token")

	t.Logf("Created root user ID: %d", userID)
	t.Logf("Access token: %s...", accessToken[:20])
	t.Logf("Refresh token: %s...", refreshToken[:20])
}
