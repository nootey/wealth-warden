package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"wealth-warden/internal/runtime"
	"wealth-warden/pkg/config"
)

var httpServerCmd = &cobra.Command{
	Use:   "http",
	Short: "Run the API server",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		logger := ctx.Value("logger").(*zap.Logger)

		cfg, err := config.LoadConfig(nil)
		if err != nil {
			logger.Fatal("Failed to load configuration: ", zap.Error(err))
		}

		logger.Info("Configuration loaded",
			zap.String("port", cfg.HttpServer.Port),
			zap.String("database", cfg.MySQL.Database),
			zap.Bool("release", cfg.Release),
		)

		app := runtime.NewServerRuntime(cfg, logger)
		return app.Run(ctx)
	},
}
