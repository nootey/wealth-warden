package seeders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database/seeders/workers"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SeederFunc func(ctx context.Context, db *gorm.DB, cfg *config.Config) error

type SeederWorkers struct {
	Func  SeederFunc
	Basic bool
	Full  bool
}

func clearStorage() error {
	storagePath := "./storage"

	// Whitelist of folders to preserve
	whitelist := []string{
		"mailer-templates",
		"migrations",
	}

	entries, err := os.ReadDir(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {

		// Skip whitelisted folders
		if slices.Contains(whitelist, entry.Name()) {
			continue
		}
		err = os.RemoveAll(filepath.Join(storagePath, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func SeedDatabase(ctx context.Context, db *gorm.DB, logger *zap.Logger, cfg *config.Config, seederType string, seederName ...string) error {
	var seeders []SeederFunc

	allSeeders := map[string]SeederWorkers{
		"SeedDefaultSettings":     {Func: workers.SeedDefaultSettings, Basic: true, Full: true},
		"SeedRolesAndPermissions": {Func: workers.SeedRolesAndPermissions, Basic: true, Full: true},
		"SeedRootUser":            {Func: workers.SeedRootUser, Basic: true, Full: true},
		"SeedMemberUser":          {Func: workers.SeedMemberUser, Basic: false, Full: true},
		"SeedAccountTypes":        {Func: workers.SeedAccountTypes, Basic: true, Full: true},
		"SeedAccounts":            {Func: workers.SeedAccounts, Basic: false, Full: true},
		"SeedCategories":          {Func: workers.SeedCategories, Basic: true, Full: true},
		"SeedTransactions":        {Func: workers.SeedTransactions, Basic: false, Full: true},
	}

	switch seederType {
	case "full":
		if err := clearStorage(); err != nil {
			return err
		}
		for _, worker := range allSeeders {
			if worker.Full {
				seeders = append(seeders, worker.Func)
			}
		}
	case "basic":
		if err := clearStorage(); err != nil {
			return err
		}
		for _, worker := range allSeeders {
			if worker.Basic {
				seeders = append(seeders, worker.Func)
			}
		}
	case "individual":
		if len(seederName) == 0 {
			return fmt.Errorf("seeder name required for individual seeder type")
		}
		worker, ok := allSeeders[seederName[0]]
		if !ok {
			return fmt.Errorf("unknown seeder: %s", seederName[0])
		}
		seeders = []SeederFunc{worker.Func}
	default:
		return fmt.Errorf("unknown seeder type: %s", seederType)
	}

	// Execute all seeders within a transaction
	err := db.Transaction(func(tx *gorm.DB) error {
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
