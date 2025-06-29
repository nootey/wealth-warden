package main

import (
	"context"
	"fmt"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [type]",
	Short: "Run database migrations",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		logger := ctx.Value("logger").(*zap.Logger)
		migrationType := "help"

		if len(args) > 0 {
			migrationType = args[0]
		}

		return runMigrations(migrationType, logger)
	},
}

func runMigrations(migrationType string, logger *zap.Logger) error {

	logger.Info("Starting database migrations")

	cfg := config.LoadConfig()
	logger.Info("Loaded the configuration", zap.Any("config", cfg))

	// Ensure the database exists before migrating
	if err := database.EnsureDatabaseExists(cfg); err != nil {
		return fmt.Errorf("database check failed: %v", err)
	}

	// Connect to MySQL using GORM
	gormDB, err := database.ConnectToMySQL(cfg, true)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %v", err)
	}

	// Get the raw *sql.DB from GORM
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw SQL DB: %v", err)
	}

	migrationsDir := "./pkg/database/migrations"
	goose.SetDialect("mysql")

	switch migrationType {
	case "up":
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			return fmt.Errorf("failed to apply migrations: %v", err)
		}
	case "down":
		if err := goose.Down(sqlDB, migrationsDir); err != nil {
			return fmt.Errorf("failed to rollback migrations: %v", err)
		}
	case "status":
		if err := goose.Status(sqlDB, migrationsDir); err != nil {
			return fmt.Errorf("failed to get migration status: %v", err)
		}
	case "fresh", "fresh-seed-full", "fresh-seed-basic":

		// Drop all tables explicitly.
		if err := database.DropAndRecreateDatabase(sqlDB, cfg); err != nil {
			return fmt.Errorf("failed to recreate database: %v", err)
		}

		// Then, run migrations.
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			return fmt.Errorf("failed to apply fresh migrations: %v", err)
		}

		// If seeding is required, run the seeder.
		if migrationType == "fresh-seed-full" || migrationType == "fresh-seed-basic" {
			seedType := "full"
			if migrationType == "fresh-seed-basic" {
				seedType = "basic"
			}
			ctx := context.Background()
			if err := seeders.SeedDatabase(ctx, gormDB, seedType); err != nil {
				return fmt.Errorf("failed to seed database: %v", err)
			}
		}
	case "help":
		return fmt.Errorf("\n Provide an additional argument to the migration function. Valid arguments are: up, down, status, fresh")
	default:
		return fmt.Errorf("invalid migration type: %s", migrationType)
	}

	logger.Info("Migrations completed successfully")
	return nil
}
