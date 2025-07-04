package main

import (
	"context"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"wealth-warden/pkg/config"
	logging "wealth-warden/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: "0.1.0",
}

func init() {
	rootCmd.AddCommand(httpServerCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}

func Execute() {

	// Load config
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Define logger and pass it into cobra commands
	logger := logging.InitLogger(cfg.Release)
	defer logger.Sync()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "logger", logger)
	ctx = context.WithValue(ctx, "config", cfg)
	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute root command", zap.Error(err))
	}
}
