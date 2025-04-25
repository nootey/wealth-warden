package main

import (
	"context"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"os/signal"
	"syscall"
	serverHttp "wealth-warden/internal/http"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
)

func main() {
	Execute()
}

func init() {
	// Initialize cobra and add the commands
	cobra.OnInitialize()

	// Register commands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
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

	//disableDBLogging := !cfg.Release
	dbClient, err := database.ConnectToMySQL(cfg, true)
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
