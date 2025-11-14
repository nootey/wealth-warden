package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders/workers"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"testing"
)

var (
	testDB        *gorm.DB
	testLogger    *zap.Logger
	testCfg       *config.Config
	testContainer *bootstrap.Container
)

func TestMain(m *testing.M) {
	var err error

	configPath := filepath.Join("..", "pkg", "config")
	testCfg, err = config.LoadConfig(&configPath, "test")
	if err != nil {
		panic(fmt.Sprintf("Failed to load test config: %v", err))
	}

	// Override database name for testing
	testCfg.Postgres.Database = "wealth_warden_test"

	testLogger = zap.NewNop() // Silent logger for tests

	// Ensure test database exists
	if err := database.EnsureDatabaseExists(testCfg); err != nil {
		panic(fmt.Sprintf("Failed to ensure test DB exists: %v", err))
	}

	testDB, err = database.ConnectToPostgres(testCfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test DB: %v", err))
	}

	// Run migrations
	if err := runTestMigrations(); err != nil {
		panic(fmt.Sprintf("Failed to run migrations: %v", err))
	}

	// Run basic seeders
	ctx := context.Background()
	if err := seedReferenceData(ctx); err != nil {
		panic(fmt.Sprintf("Failed to seed reference data: %v", err))
	}

	testContainer, err = bootstrap.NewContainer(testCfg, testDB, testLogger)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test container: %v", err))
	}

	// Run all tests
	code := m.Run()

	cleanupTestDB()
	os.Exit(code)
}

func runTestMigrations() error {
	sqlDB, err := testDB.DB()
	if err != nil {
		return err
	}

	migrationsDir := filepath.Join("..", "pkg", "database", "migrations")
	err = goose.SetDialect("postgres")
	if err != nil {
		return err
	}

	_ = goose.Reset(sqlDB, migrationsDir)

	return goose.Up(sqlDB, migrationsDir)
}

func seedReferenceData(ctx context.Context) error {
	seeders := []func(context.Context, *gorm.DB, *zap.Logger, *config.Config) error{
		workers.SeedDefaultSettings,
		workers.SeedRolesAndPermissions,
		workers.SeedAccountTypes,
		workers.SeedCategories,
	}

	for _, seeder := range seeders {
		if err := seeder(ctx, testDB, testLogger, testCfg); err != nil {
			return err
		}
	}
	return nil
}

func cleanupTestDB() {
	if testDB != nil {
		sqlDB, _ := testDB.DB()
		err := sqlDB.Close()
		if err != nil {
			fmt.Println("failed to close test DB")
		}
	}
}
