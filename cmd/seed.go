package main

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var seedCmd = &cobra.Command{
	Use:   "seed [type] [name]",
	Short: "Run database seeders",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {

		seedType := "help"
		seedLogger := logger.Named("seeder")
		seedLogger.Info("Loaded the configuration", zap.Any("config", cfg))

		if len(args) > 0 {
			seedType = args[0]
		}

		var seederName string
		if len(args) > 1 {
			seederName = args[1]
		}

		return runSeeders(seedType, seederName, cfg, seedLogger)
	},
}

var validSeedTypes = map[string]bool{
	"full":       true,
	"basic":      true,
	"individual": true,
	"help":       true,
}

func isValidSeedType(seedType string) bool {
	return validSeedTypes[seedType]
}

func runSeeders(seedType string, seederName string, cfg *config.Config, logger *zap.Logger) error {

	if !isValidSeedType(seedType) {
		return fmt.Errorf("invalid seed type: %s", seedType)
	}

	if seedType == "help" {
		return fmt.Errorf("\n Provide an additional argument to the seeder function. Valid arguments are: full, basic, individual <name>")
	}

	if seedType == "individual" && seederName == "" {
		return fmt.Errorf("seeder name required for individual type")
	}

	logger.Info("Starting database seeding")

	dbLogger := logger.Named("database")
	gormDB, err := database.ConnectToPostgres(cfg, dbLogger)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{}
	if seederName != "" {
		args = append(args, seederName)
	}

	if err := seeders.SeedDatabase(ctx, gormDB, logger, cfg, seedType, args...); err != nil {
		return fmt.Errorf("failed to seed database: %v", err)
	}

	return nil
}
