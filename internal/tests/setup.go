// internal/tests/setup.go
package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders/workers"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB        *gorm.DB
	Logger    *zap.Logger
	Config    *config.Config
	Container *bootstrap.Container
)

var (
	setupOnce sync.Once
	setupErr  error
)

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func Setup() error {
	setupOnce.Do(func() {
		setupErr = doSetup()
	})
	return setupErr
}

func doSetup() error {

	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, "pkg", "config")

	cfg, err := config.LoadConfig(&configPath, "test")
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}

	cfg.Postgres.Database = "wealth_warden_test"
	Config = cfg
	Logger = zap.NewNop()

	if err := database.EnsureDatabaseExists(Config); err != nil {
		return fmt.Errorf("failed to ensure test DB exists: %w", err)
	}

	DB, err = database.ConnectToPostgres(Config)
	if err != nil {
		return fmt.Errorf("failed to connect to test DB: %w", err)
	}

	if err := runMigrations(projectRoot); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	ctx := context.Background()
	if err := seedReferenceData(ctx); err != nil {
		return fmt.Errorf("failed to seed reference data: %w", err)
	}

	Container, err = bootstrap.NewContainer(Config, DB, Logger)
	if err != nil {
		return fmt.Errorf("failed to create test container: %w", err)
	}

	if err := SetupRootUser(); err != nil {
		return fmt.Errorf("failed to setup root user: %w", err)
	}

	return nil
}

func Teardown() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		_ = sqlDB.Close()
	}
}

func CleanupData(t *testing.T) {
	t.Helper()

	tables := []string{
		"category_group_members",
		"category_groups",
		"exports",
		"transaction_templates",
		"imports",
		"tokens",
		"invitations",
		"account_daily_snapshots",
		"settings_user",
		"settings_general",
		"activity_logs",
		"transfers",
		"transactions",
		"balances",
		"accounts",
	}

	for _, table := range tables {
		result := DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if result.Error != nil {
			t.Logf("Error deleting from %s: %v", table, result.Error)
		}
	}
}

func runMigrations(projectRoot string) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	if err := cleanDatabase(); err != nil {
		return fmt.Errorf("failed to clean database: %w", err)
	}

	migrationsDir := filepath.Join(projectRoot, "pkg", "database", "migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(sqlDB, migrationsDir)
}

func cleanDatabase() error {
	if err := DB.Exec(`
        DO $$ 
        DECLARE 
            r RECORD;
        BEGIN
            FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
                EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
            END LOOP;
            
            FOR r IN (SELECT typname FROM pg_type WHERE typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public') AND typtype = 'e') LOOP
                EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
            END LOOP;
            
            DROP TABLE IF EXISTS goose_db_version CASCADE;
        END $$;
    `).Error; err != nil {
		return err
	}

	return nil
}

func seedReferenceData(ctx context.Context) error {
	seeders := []func(context.Context, *gorm.DB, *zap.Logger, *config.Config) error{
		workers.SeedDefaultSettings,
		workers.SeedRolesAndPermissions,
		workers.SeedAccountTypes,
		workers.SeedCategories,
	}

	for _, seeder := range seeders {
		if err := seeder(ctx, DB, Logger, Config); err != nil {
			return err
		}
	}
	return nil
}
