package runtime

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os/signal"
	"syscall"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
)

type ServerRuntime struct {
	Logger *zap.Logger
	Config *config.Config
}

func NewServerRuntime(cfg *config.Config, logger *zap.Logger) *ServerRuntime {
	return &ServerRuntime{
		Config: cfg,
		Logger: logger,
	}
}

func (rt *ServerRuntime) Run(context context.Context) error {
	ctx, stop := signal.NotifyContext(context, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dbClient, err := database.ConnectToMySQL(rt.Config, true)
	if err != nil {
		rt.Logger.Error("Database connection failed", zap.Error(err))
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer database.DisconnectMySQL()
	rt.Logger.Info("Successfully connected to the database")

	httpLogger := rt.Logger.Named("http").With(zap.String("component", "HTTP"))
	container := bootstrap.NewContainer(rt.Config, dbClient, httpLogger)
	httpServer := http.NewServer(container, httpLogger)
	go httpServer.Start()

	<-ctx.Done()

	rt.Logger.Info("Interrupt signal received, shutting down HTTP server...")
	if err := httpServer.Shutdown(); err != nil {
		rt.Logger.Error("HTTP server shutdown failed", zap.Error(err))
		return fmt.Errorf("http server shutdown failed: %w", err)
	}
	rt.Logger.Info("HTTP server exiting")
	return nil
}
