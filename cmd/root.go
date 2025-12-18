package main

import (
	"fmt"
	"wealth-warden/pkg/config"
	logging "wealth-warden/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// Version info (injected at build time via ldflags)
	Version   = "dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"

	cfg    *config.Config
	logger *zap.Logger
)
var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, CommitSHA, BuildTime),
}

func init() {
	rootCmd.AddCommand(httpServerCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}

func Execute() {

	var err error

	// Load config
	cfg, err = config.LoadConfig(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
	}

	// Define logger and pass it into cobra commands
	logger = logging.InitLogger(cfg.Release)
	defer func() {
		_ = logger.Sync() // Ignore sync errors
	}()

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute root command", zap.Error(err))
	}
}
