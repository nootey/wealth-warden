package main

import (
	"context"
	"fmt"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [type]",
	Short: "Run database migrations",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		migrationType := "help"

		if len(args) > 0 {
			migrationType = args[0]
		}

		migrationDir, err := cmd.Flags().GetString("dir")
		if err != nil {
			return fmt.Errorf("failed to get migration directory flag: %v", err)
		}

		mgLogger := logger.Named("migrate")
		return runMigrations(migrationType, migrationDir, cfg, mgLogger)
	},
}

func init() {
	migrateCmd.Flags().StringP("dir", "d", "./storage/migrations", "Migration directory path")
}

func runMigrations(migrationType string, migrationsDir string, cfg *config.Config, logger *zap.Logger) error {
	logger.Info("Starting database migrations", zap.String("migration_dir", migrationsDir))
	dbLogger := logger.Named("database")
	// Ensure the database exists before migrating
	if err := database.EnsureDatabaseExists(cfg, dbLogger); err != nil {
		return fmt.Errorf("database check failed: %v", err)
	}

	gormDB, err := database.ConnectToPostgres(cfg, dbLogger)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw SQL DB: %v", err)
	}

	err = goose.SetDialect("postgres")
	if err != nil {
		return err
	}

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

		// Close the current connection because it connects to the target database.
		err := sqlDB.Close()
		if err != nil {
			return err
		}

		// Connect to a maintenance database (like "postgres").
		mDB, err := database.ConnectToMaintenance(cfg, dbLogger)
		if err != nil {
			return fmt.Errorf("failed to connect to maintenance database: %v", err)
		}
		mSqlDB, err := mDB.DB()
		if err != nil {
			return fmt.Errorf("failed to get raw SQL DB for maintenance: %v", err)
		}

		// Drop all tables explicitly.
		if err := database.DropAndRecreateDatabase(mSqlDB, cfg); err != nil {
			return fmt.Errorf("failed to recreate database: %v", err)
		}
		err = mSqlDB.Close()
		if err != nil {
			return err
		} // Close the maintenance connection.

		// Reconnect to the newly created target database.
		gormDB, err = database.ConnectToPostgres(cfg, dbLogger)
		if err != nil {
			return fmt.Errorf("failed to reconnect to Postgres: %v", err)
		}
		sqlDB, err = gormDB.DB()
		if err != nil {
			return fmt.Errorf("failed to get raw SQL DB after reconnection: %v", err)
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
			if err := seeders.SeedDatabase(ctx, gormDB, logger, cfg, seedType); err != nil {
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
