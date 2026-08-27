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
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/worker"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/telemetry"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type App struct {
	logger    *zap.Logger
	http      *http.HttpServer
	scheduler *jobscheduler.Scheduler
	jobWorker *jobworker.Service
	telemetry *telemetry.Provider
	health    *health.Service
	hub       *ws.Hub
	redis     *redis.Client
	riverPool *pgxpool.Pool
}

func New(cfg *config.Config, logger *zap.Logger) (*App, error) {

	// Database
	dbClient, err := database.ConnectToPostgres(cfg, logger.Named("database"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Redis (session store)
	redisClient, err := database.ConnectToRedis(cfg, logger.Named("redis"))
	if err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	// The dispatcher and workers register OTEL instruments against the global providers set here.
	tel, err := telemetry.New(context.Background(), cfg.Otel, logger.Named("telemetry"))
	if err != nil {
		return nil, fmt.Errorf("telemetry initialization failed: %w", err)
	}

	riverPool, err := database.ConnectRiverPool(context.Background(), cfg, logger.Named("river-pool"))
	if err != nil {
		return nil, fmt.Errorf("river pool connection failed: %w", err)
	}
	// Only Run() gets to close the pool, and only once New has handed back an App.
	started := false
	defer func() {
		if !started {
			riverPool.Close()
		}
	}()

	riverClient, riverWorkers, err := jobworker.NewClient(riverPool, logger.Named("job-worker"), cfg.Queue, cfg.Otel.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("river client initialization failed: %w", err)
	}

	meter := otel.GetMeterProvider().Meter(cfg.Otel.ServiceName)

	jobDispatcher, err := queue.NewRiverDispatcher(riverClient, meter)
	if err != nil {
		return nil, fmt.Errorf("job dispatcher initialization failed: %w", err)
	}

	container, err := bootstrap.NewServiceContainer(cfg, dbClient, redisClient, logger.Named("container"), jobDispatcher, nil)
	if err != nil {
		return nil, fmt.Errorf("container initialization failed: %w", err)
	}

	if err := jobworker.RegisterWorkers(riverWorkers, container, logger.Named("job-worker")); err != nil {
		return nil, fmt.Errorf("job worker registration failed: %w", err)
	}

	if err := jobworker.RegisterQueueLagGauge(repositories.NewJobRepository(dbClient), meter); err != nil {
		return nil, fmt.Errorf("queue lag gauge registration failed: %w", err)
	}

	scheduler, err := jobscheduler.NewScheduler(logger.Named("scheduler"), container, jobscheduler.FlagsFromConfig(cfg.Scheduler), cfg.Scheduler.ConcurrentWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
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
	healthSvc.Add(health.NewRedisChecker(redisClient))

	started = true
	return &App{
		logger:    logger,
		http:      http.NewServer(container, logger.Named("http"), healthSvc.Handler()),
		scheduler: scheduler,
		jobWorker: jobworker.NewService(riverClient, logger.Named("job-worker")),
		telemetry: tel,
		health:    healthSvc,
		hub:       container.Hub,
		redis:     redisClient,
		riverPool: riverPool,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		_ = database.DisconnectPostgres()
	}()
	defer func() {
		_ = a.redis.Close()
	}()
	defer a.riverPool.Close()
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

	supervisor.Add(worker.NewService("job-worker", a.jobWorker.Run))

	supervisor.Add(worker.NewService("health", a.health.Run))

	supervisor.Add(worker.NewService("ws-hub", a.hub.Run))

	supervisor.Run(ctx)
	return nil
}
