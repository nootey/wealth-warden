package app

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/worker"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/telemetry"

	"go.uber.org/zap"
)

type App struct {
	logger       *zap.Logger
	http         *http.HttpServer
	scheduler    *jobscheduler.Scheduler
	jobQueue     *queue.JobQueue
	telemetry    *telemetry.Provider
}

func New(cfg *config.Config, logger *zap.Logger) (*App, error) {

	// Database
	dbClient, err := database.ConnectToPostgres(cfg, logger.Named("database"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// In memory job queue
	jobQueue := queue.NewJobQueue(1, 25)
	jobDispatcher := &queue.InMemoryDispatcher{Queue: jobQueue}

	container, err := bootstrap.NewServiceContainer(cfg, dbClient, logger.Named("container"), jobDispatcher, nil)
	if err != nil {
		return nil, fmt.Errorf("container initialization failed: %w", err)
	}

	scheduler, err := jobscheduler.NewScheduler(logger.Named("scheduler"), container, jobscheduler.FlagsFromConfig(cfg.Scheduler), cfg.Scheduler.ConcurrentWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	tel, err := telemetry.New(context.Background(), cfg.Otel, logger.Named("telemetry"))
	if err != nil {
		return nil, fmt.Errorf("telemetry initialization failed: %w", err)
	}

	return &App{
		logger:    logger,
		http:      http.NewServer(container, logger.Named("http")),
		scheduler: scheduler,
		jobQueue:  jobQueue,
		telemetry: tel,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		_ = database.DisconnectPostgres()
	}()
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := a.telemetry.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("telemetry shutdown failed", zap.Error(err))
		}
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

	supervisor.Add(worker.NewService("job-queue", func(ctx context.Context) {
		a.jobQueue.Run(ctx)
		if err := a.jobQueue.Shutdown(); err != nil {
			a.logger.Error("job queue shutdown failed", zap.Error(err))
			cancel()
		}
	}))

	supervisor.Run(ctx)
	return nil
}
