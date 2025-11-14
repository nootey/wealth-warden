package main

import (
	"context"
	"wealth-warden/pkg/config"
	logging "wealth-warden/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type contextKey string

const (
	loggerKey contextKey = "ww_logger"
	configKey contextKey = "ww_config"
)

var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: "1.0.0",
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
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(logger)

	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, logger)
	ctx = context.WithValue(ctx, configKey, cfg)
	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute root command", zap.Error(err))
	}
}
