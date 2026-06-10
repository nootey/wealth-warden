package app

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/health"
	"wealth-warden/internal/http"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/jobworker"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/worker"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/telemetry"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type App struct {
	logger    *zap.Logger
	http      *http.HttpServer
	scheduler *jobscheduler.Scheduler
	consumer  *jobworker.Consumer
	telemetry *telemetry.Provider
	health    *health.Service
}

func New(cfg *config.Config, logger *zap.Logger) (*App, error) {

	// Database
	dbClient, err := database.ConnectToPostgres(cfg, logger.Named("database"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Telemetry: the dispatcher and consumer register OTEL instruments against the global providers set here.
	tel, err := telemetry.New(context.Background(), cfg.Otel, logger.Named("telemetry"))
	if err != nil {
		return nil, fmt.Errorf("telemetry initialization failed: %w", err)
	}

	// DB-backed job queue
	jobDispatcher, err := queue.NewDBDispatcher(dbClient, otel.GetMeterProvider().Meter(cfg.Otel.ServiceName))
	if err != nil {
		return nil, fmt.Errorf("job dispatcher initialization failed: %w", err)
	}

	container, err := bootstrap.NewServiceContainer(cfg, dbClient, logger.Named("container"), jobDispatcher, nil)
	if err != nil {
		return nil, fmt.Errorf("container initialization failed: %w", err)
	}

	scheduler, err := jobscheduler.NewScheduler(logger.Named("scheduler"), container, jobscheduler.FlagsFromConfig(cfg.Scheduler), cfg.Scheduler.ConcurrentWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	consumer, err := jobworker.NewConsumer(container, logger.Named("job-consumer"), cfg.Queue)
	if err != nil {
		return nil, fmt.Errorf("failed to create job consumer: %w", err)
	}

	healthSvc, err := health.New(logger.Named("health"))
	if err != nil {
		return nil, fmt.Errorf("health service initialization failed: %w", err)
	}
	sqlDB, err := dbClient.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB for health check: %w", err)
	}
	healthSvc.Add(health.NewDBChecker(sqlDB))

	return &App{
		logger:    logger,
		http:      http.NewServer(container, logger.Named("http"), healthSvc.Handler()),
		scheduler: scheduler,
		consumer:  consumer,
		telemetry: tel,
		health:    healthSvc,
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

	supervisor.Add(worker.NewService("job-consumer", a.consumer.Run))

	supervisor.Add(worker.NewService("health", a.health.Run))

	supervisor.Run(ctx)
	return nil
}
