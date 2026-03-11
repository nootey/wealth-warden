package main

import (
	"wealth-warden/internal/runtime"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Run the app, with workers for http server, scheduler ...",
	RunE: func(cmd *cobra.Command, args []string) error {

		logger.Info("Configuration loaded",
			zap.String("database", cfg.Postgres.Database),
			zap.Bool("release", cfg.Release),
		)

		app := runtime.NewAppRuntime(cfg, logger)
		return app.Run(cmd.Context())
	},
}
