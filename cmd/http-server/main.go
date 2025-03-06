package main

import (
	"context"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"os/signal"
	"syscall"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"
	serverHttp "wealth-warden/pkg/http"
)

// rootCmd is the main entry point for the app
var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: "1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// migrateCmd handles running stacked migrations
var migrateCmd = &cobra.Command{
	Use:   "migrate [type]",
	Short: "Run database migrations",
	Args:  cobra.MaximumNArgs(1), // Ensure only one argument like "up", "down", etc.
	Run: func(cmd *cobra.Command, args []string) {
		migrationType := "help"

		if len(args) > 0 {
			migrationType = args[0]
		}

		runMigrations(migrationType)
	},
}

// seedCmd handles seeding  tables
var seedCmd = &cobra.Command{
	Use:   "seed [type]",
	Short: "Run database seeders",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		seedType := "help"

		if len(args) > 0 {
			seedType = args[0]
		}

		runSeeders(seedType)
	},
}

func init() {
	// Initialize cobra and add the commands
	cobra.OnInitialize()

	// Register commands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		zap.L().Fatal("Failed to execute the root command", zap.Error(err))
	}
}

func runServer() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
	defer logger.Sync()

	cfg := config.LoadConfig()
	logger.Info("Loaded the configuration", zap.Any("config", cfg))

	// Ensure the database exists before connecting
	if err := database.EnsureDatabaseExists(cfg); err != nil {
		log.Fatalf("Database check failed: %v", err)
	}

	dbClient, err := database.ConnectToMySQL(cfg, !cfg.Release)
	if err != nil {
		log.Fatalf("MySQL Connection Error: %v", err)
	}
	defer database.DisconnectMySQL()

	// Initialize the server with the logger
	httpServer := serverHttp.NewServer(cfg, logger, dbClient)
	go httpServer.Start()

	// Wait for the interrupt signal
	<-ctx.Done()

	// Gracefully shutdown the HTTP server
	logger.Info("Shutting down server...")
	if err := httpServer.Shutdown(); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}

func runMigrations(migrationType string) {
	// Initialize the logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("Starting database migrations")

	// Load Configuration
	cfg := config.LoadConfig()
	logger.Info("Loaded the configuration", zap.Any("config", cfg))

	// Connect to MySQL using GORM
	gormDB, err := database.ConnectToMySQL(cfg, true)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Get the raw *sql.DB from GORM
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Failed to get raw SQL DB: %v", err)
	}

	migrationsDir := "./pkg/database/migrations"
	goose.SetDialect("mysql")

	switch migrationType {
	case "up":
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to apply migrations: %v", err)
		}
	case "down":
		if err := goose.Down(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
	case "status":
		if err := goose.Status(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	case "fresh":
		if err := goose.Reset(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to reset migrations: %v", err)
		}
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to apply fresh migrations: %v", err)
		}
	case "fresh-seed-full", "fresh-seed-basic":
		// Extract seed type from the migrationType (last part after "fresh-seed-")
		seedType := "full"
		if migrationType == "fresh-seed-basic" {
			seedType = "basic"
		}

		// Reset database and reapply migrations
		if err := goose.Reset(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to reset migrations: %v", err)
		}
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to apply fresh migrations: %v", err)
		}

		// Run the seeder
		ctx := context.Background()
		if err := seeders.SeedDatabase(ctx, gormDB, seedType); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	case "help":
		log.Fatal("\n Provide an additional argument to the migration function. Valid arguments are: up, down, status, fresh")
	default:
		log.Fatalf("Invalid migration type: %s", migrationType)
	}

	logger.Info("Migrations completed successfully")
}

func runSeeders(seedType string) {

	// Validate seed type
	var validSeedTypes = map[string]bool{
		"full":  true,
		"basic": true,
		"help":  true,
	}

	if !validSeedTypes[seedType] {
		log.Fatalf("Invalid seed type provided: %s.", seedType)
	}

	// Initialize the logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("Starting database seeding")

	// Load Configuration
	cfg := config.LoadConfig()
	logger.Info("Loaded the configuration", zap.Any("config", cfg))

	// Connect to MySQL using GORM
	gormDB, err := database.ConnectToMySQL(cfg, false)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	ctx := context.Background()

	switch seedType {
	case "full":
		err = seeders.SeedDatabase(ctx, gormDB, "full")
		if err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	case "basic":
		err = seeders.SeedDatabase(ctx, gormDB, "basic")
		if err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	case "help":
		log.Fatal("\n Provide an additional argument to the seeder function. Valid arguments are: full, basic")
	default:
		log.Fatalf("Invalid seeder type: %s", seedType)
	}

}
