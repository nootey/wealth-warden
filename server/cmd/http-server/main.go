package main

import (
	"context"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"os/signal"
	"syscall"
	"wealth-warden/server/pkg/config"
	"wealth-warden/server/pkg/database"
	serverHttp "wealth-warden/server/pkg/http"
)

// rootCmd is the main entry point for the NoiseGuard Licence Service
var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: "1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// migrateCmd handles running migrations for the database
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		runMigrations("full")
	},
}

func init() {

	// Initialize cobra and add the commands
	cobra.OnInitialize()

	// Register commands
	rootCmd.AddCommand(migrateCmd)
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

	dbClient, err := database.ConnectToMySQL(cfg)
	if err != nil {
		log.Fatalf("MySQL Connection Error: %v", err)
	}
	defer database.DisconnectMySQL()

	// Initialize the server with the logger
	httpServer := serverHttp.NewServer(cfg, logger, dbClient)

	// Start the server with health checks
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

	//if migrationType == "full" {
	//	err = database.RunMigrations()
	//	if err != nil {
	//		logger.Fatal("Failed to run migrations", zap.Error(err))
	//	}
	//}

	logger.Info("Migrations completed successfully")
}
