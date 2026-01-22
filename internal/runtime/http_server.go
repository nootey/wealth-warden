package runtime

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"

	"go.uber.org/zap"
)

type HttpServerRuntime struct {
	Logger *zap.Logger
	Config *config.Config
}

func NewHttpServerRuntime(cfg *config.Config, logger *zap.Logger) *HttpServerRuntime {
	return &HttpServerRuntime{
		Config: cfg,
		Logger: logger,
	}
}

func (rt *HttpServerRuntime) Run(context context.Context) error {
	ctx, stop := signal.NotifyContext(context, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Connect to DB
	dbLogger := rt.Logger.Named("database")
	dbClient, err := database.ConnectToPostgres(rt.Config, dbLogger)
	if err != nil {
		dbLogger.Error("Database connection failed", zap.Error(err))
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() {
		err := database.DisconnectPostgres()
		if err != nil {
			dbLogger.Error("Failed to disconnect postgres cleanly", zap.Error(err))
		}
	}()
	dbLogger.Info("Successfully connected to the database")

	// Initialize container
	containerLogger := rt.Logger.Named("container")
	httpLogger := rt.Logger.Named("http")
	container, err := bootstrap.NewServiceContainer(rt.Config, dbClient, containerLogger)
	if err != nil {
		containerLogger.Error("Container initialization failed", zap.Error(err))
		return fmt.Errorf("cntainer initialization failed: %w", err)
	}

	// Start scheduler
	schedulerLogger := rt.Logger.Named("scheduler")
	scheduler, err := NewScheduler(schedulerLogger, container, SchedulerConfig{
		StartBackfillImmediately:  false,
		StartTemplateImmediately:  false,
		StartPriceSyncImmediately: true,
	})
	if err != nil {
		schedulerLogger.Error("Failed to create scheduler", zap.Error(err))
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	if err := scheduler.Start(); err != nil {
		schedulerLogger.Error("Failed to start scheduler", zap.Error(err))
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	defer func(scheduler *Scheduler) {
		err := scheduler.Shutdown()
		if err != nil {
			panic("failed to shutdown scheduler")
		}
	}(scheduler)

	// Start HTTP server
	httpServer := http.NewServer(container, httpLogger)
	go httpServer.Start()

	<-ctx.Done()

	httpLogger.Info("Interrupt signal received, shutting down HTTP server...")
	if err := httpServer.Shutdown(); err != nil {
		httpLogger.Error("HTTP server shutdown failed", zap.Error(err))
		return fmt.Errorf("http server shutdown failed: %w", err)
	}
	httpLogger.Info("HTTP server exiting")
	return nil
}
