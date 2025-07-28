package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"
)

var seedCmd = &cobra.Command{
	Use:   "seed [type]",
	Short: "Run database seeders",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		logger := ctx.Value("logger").(*zap.Logger)
		seedType := "help"
		cfg := ctx.Value("config").(*config.Config)
		logger.Info("Loaded the configuration", zap.Any("config", cfg))

		if len(args) > 0 {
			seedType = args[0]
		}

		return runSeeders(seedType, cfg, logger)
	},
}

var validSeedTypes = map[string]bool{
	"full":  true,
	"basic": true,
	"help":  true,
}

func isValidSeedType(seedType string) bool {
	return validSeedTypes[seedType]
}

func runSeeders(seedType string, cfg *config.Config, logger *zap.Logger) error {

	if !isValidSeedType(seedType) {
		return fmt.Errorf("invalid seed type: %s", seedType)
	}

	logger.Info("Starting database seeding")

	gormDB, err := database.ConnectToPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	ctx := context.Background()

	switch seedType {
	case "full":
		err = seeders.SeedDatabase(ctx, gormDB, logger, "full")
		if err != nil {
			return fmt.Errorf("failed to seed database: %v", err)
		}
	case "basic":
		err = seeders.SeedDatabase(ctx, gormDB, logger, "basic")
		if err != nil {
			return fmt.Errorf("failed to seed database: %v", err)
		}
	case "help":
		return fmt.Errorf("\n Provide an additional argument to the seeder function. Valid arguments are: full, basic")
	default:
		return fmt.Errorf("invalid seeder type: %s", seedType)
	}
	return nil
}
