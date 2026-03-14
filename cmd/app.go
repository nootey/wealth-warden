package main

import (
	"os/signal"
	"syscall"
	"wealth-warden/internal/app"

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

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()

		a, err := app.New(cfg, logger)
		if err != nil {
			return err
		}
		return a.Run(ctx)
	},
}
