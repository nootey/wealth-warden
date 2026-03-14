package app

import (
	"context"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/worker"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"

	"go.uber.org/zap"
)

type App struct {
	logger    *zap.Logger
	http      *http.HttpServer
	scheduler *jobscheduler.Scheduler
}

func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	dbClient, err := database.ConnectToPostgres(cfg, logger.Named("database"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	container, err := bootstrap.NewServiceContainer(cfg, dbClient, logger.Named("container"))
	if err != nil {
		return nil, fmt.Errorf("container initialization failed: %w", err)
	}

	scheduler, err := jobscheduler.NewScheduler(logger.Named("scheduler"), container, jobscheduler.SchedulerConfig{
		StartBackfillImmediately:  false,
		StartTemplateImmediately:  false,
		StartPriceSyncImmediately: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	return &App{
		logger:    logger,
		http:      http.NewServer(container, logger.Named("http")),
		scheduler: scheduler,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		_ = database.DisconnectPostgres()
	}()

	supervisor := worker.NewSupervisor(a.logger.Named("supervisor"))

	supervisor.Add(worker.NewService("http", func(ctx context.Context) {
		if err := a.http.Start(ctx); err != nil {
			a.logger.Error("http server failed", zap.Error(err))
			cancel()
		}
		_ = a.http.Shutdown()
	}))

	supervisor.Add(worker.NewService("scheduler", func(ctx context.Context) {
		if err := a.scheduler.Start(ctx); err != nil {
			a.logger.Error("scheduler failed", zap.Error(err))
			cancel()
		}
		_ = a.scheduler.Shutdown()
	}))

	supervisor.Run(ctx)
	return nil
}
