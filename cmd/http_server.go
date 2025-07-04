package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"wealth-warden/internal/models"
	"wealth-warden/internal/runtime"
)

var httpServerCmd = &cobra.Command{
	Use:   "http",
	Short: "Run the API server",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		logger := ctx.Value("logger").(*zap.Logger)
		cfg := ctx.Value("config").(*models.Config)

		logger.Info("Configuration loaded",
			zap.String("port", cfg.HttpServer.Port),
			zap.String("database", cfg.MySQL.Database),
			zap.Bool("release", cfg.Release),
		)

		app := runtime.NewServerRuntime(cfg, logger)
		return app.Run(ctx)
	},
}
