package seeders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database/seeders/workers"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SeederFunc func(ctx context.Context, db *gorm.DB, cfg *config.Config) error

func clearStorage() error {
	storagePath := "./storage"

	entries, err := os.ReadDir(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {

		// Skip the mailer-template folder
		if entry.Name() == "mailer-templates" {
			continue
		}

		err = os.RemoveAll(filepath.Join(storagePath, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func SeedDatabase(ctx context.Context, db *gorm.DB, logger *zap.Logger, cfg *config.Config, seederType string) error {
	var seeders []SeederFunc

	err := clearStorage()
	if err != nil {
		return err
	}

	switch seederType {
	case "full":

		seeders = []SeederFunc{
			workers.SeedDefaultSettings,
			workers.SeedRolesAndPermissions,
			workers.SeedRootUser,
			workers.SeedMemberUser,
			workers.SeedAccountTypes,
			workers.SeedAccounts,
			workers.SeedCategories,
			workers.SeedTransactions,
		}
	case "basic":
		seeders = []SeederFunc{
			workers.SeedDefaultSettings,
			workers.SeedRolesAndPermissions,
			workers.SeedRootUser,
			workers.SeedAccountTypes,
			//workers.SeedRootAccounts,
			workers.SeedCategories,
		}
	default:
		return fmt.Errorf("unknown seeder type: %s", seederType)
	}

	// Execute all seeders within a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, seeder := range seeders {
			// Get the function name using reflection
			seederName := getFunctionName(seeder)

			// Run the seeder
			if err := seeder(ctx, tx, cfg); err != nil {
				return fmt.Errorf("seeder %s failed: %w", seederName, err)
			}

			// Print status
			logger.Info("Seeder completed",
				zap.String("timestamp", time.Now().Format("2006/01/02 15:04:05")),
				zap.String("status", "OK"),
				zap.String("seeder", seederName),
			)

		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to run seeders: %w", err)
	}

	logger.Info("Database seeding completed successfully.")
	return nil
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
