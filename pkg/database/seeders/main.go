package seeders

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"runtime"
	"time"
	"wealth-warden/pkg/database/seeders/workers"
)

type SeederFunc func(ctx context.Context, db *gorm.DB) error

func SeedDatabase(ctx context.Context, db *gorm.DB, seederType string) error {
	var seeders []SeederFunc

	switch seederType {
	case "full":
		seeders = []SeederFunc{
			workers.SeedRolesAndPermissions,
			workers.SeedSuperAdmin,
			workers.SeedMember,
			workers.SeedInflowCategoryTable,
			workers.SeedInflowTable,
			workers.SeedOutflowCategoryTable,
			workers.SeedOutflowTable,
			workers.SeedDynamicCategories,
		}
	case "basic":
		seeders = []SeederFunc{
			workers.SeedRolesAndPermissions,
			workers.SeedSuperAdmin,
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
			if err := seeder(ctx, tx); err != nil {
				return fmt.Errorf("seeder %s failed: %w", seederName, err)
			}

			// Print status
			fmt.Printf("%s OK %s\n", time.Now().Format("2006/01/02 15:04:05"), seederName)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to run seeders: %w", err)
	}

	fmt.Println("Database seeding completed successfully.")
	return nil
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
