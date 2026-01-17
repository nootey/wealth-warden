package main

import (
	"fmt"
	"wealth-warden/pkg/config"
	logging "wealth-warden/pkg/logger"
	"wealth-warden/pkg/version"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfg    *config.Config
	logger *zap.Logger
	configPath string
)

var rootCmd = &cobra.Command{
	Use:     "wealth-warden",
	Short:   "WealthWarden server",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.CommitSHA, version.BuildTime),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config for every child command before they are executed
		var err error
		cfg, err = config.LoadConfig(&configPath)
		return fmt.Errorf("failed to load configuration: %s", err.Error())
	},
}

func init() {
	rootCmd.AddCommand(httpServerCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
}

func Execute() {
	// Define logger and pass it into cobra commands
	logger = logging.InitLogger(cfg.Release)
	defer func() {
		_ = logger.Sync() // Ignore sync errors
	}()

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute root command", zap.Error(err))
	}
}
