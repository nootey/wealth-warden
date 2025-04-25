package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"wealth-warden/internal/runtime"
	"wealth-warden/pkg/config"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the API server",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context() // get context from Cobra

		logger, err := zap.NewProduction()
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer logger.Sync()

		cfg := config.LoadConfig()
		logger.Info("Configuration loaded",
			zap.String("port", cfg.HttpServerPort),
			zap.String("database", cfg.MySQLDatabase),
			zap.Bool("release", cfg.Release),
		)

		app := runtime.NewServerRuntime(cfg, logger)
		return app.Run(ctx)
	},
}
