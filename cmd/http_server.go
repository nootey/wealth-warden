package main

import (
	"wealth-warden/internal/runtime"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var httpServerCmd = &cobra.Command{
	Use:   "http",
	Short: "Run the API server",
	RunE: func(cmd *cobra.Command, args []string) error {

		logger.Info("Configuration loaded",
			zap.String("port", cfg.HttpServer.Port),
			zap.String("database", cfg.Postgres.Database),
			zap.Bool("release", cfg.Release),
		)

		app := runtime.NewHttpServerRuntime(cfg, logger)
		return app.Run(cmd.Context())
	},
}
