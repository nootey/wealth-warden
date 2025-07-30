package seeders

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"reflect"
	"runtime"
	"time"
	"wealth-warden/pkg/database/seeders/workers"
)

type SeederFunc func(ctx context.Context, db *gorm.DB, logger *zap.Logger) error

func SeedDatabase(ctx context.Context, db *gorm.DB, logger *zap.Logger, seederType string) error {
	var seeders []SeederFunc

	switch seederType {
	case "full":
		seeders = []SeederFunc{
			workers.SeedRolesAndPermissions,
			workers.SeedRootUser,
			workers.SeedMemberUser,
			workers.SeedAccountTypes,
		}
	case "basic":
		seeders = []SeederFunc{
			workers.SeedRolesAndPermissions,
			workers.SeedRootUser,
			workers.SeedAccountTypes,
		}
	default:
		return fmt.Errorf("unknown seeder type: %s", seederType)
	}

	// Execute all seeders within a transaction
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, seeder := range seeders {
			// Get the function name using reflection
			seederName := getFunctionName(seeder)

			// Run the seeder
			if err := seeder(ctx, tx, logger); err != nil {
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
