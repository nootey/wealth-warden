package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConnection(t *testing.T) {
	require.NotNil(t, testDB, "Test database should be initialized")

	sqlDB, err := testDB.DB()
	require.NoError(t, err)

	err = sqlDB.Ping()
	assert.NoError(t, err, "Should be able to ping test database")

	t.Log("Database connection works!")
}

func TestMigrationsRan(t *testing.T) {
	var exists bool
	err := testDB.Raw(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )
    `).Scan(&exists).Error

	require.NoError(t, err)
	assert.True(t, exists, "Users table should exist after migrations")

	t.Log("Migrations ran successfully!")
}
