package runtime

import (
	"context"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"

	"go.uber.org/zap"
)

type AppRuntime struct {
	Logger *zap.Logger
	Config *config.Config
}

func NewAppRuntime(cfg *config.Config, logger *zap.Logger) *AppRuntime {
	return &AppRuntime{
		Config: cfg,
		Logger: logger,
	}
}

func (rt *AppRuntime) Run(ctx context.Context) error {

	// Connect to DB
	dbClient, err := database.ConnectToPostgres(rt.Config, rt.Logger.Named("database"))
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() {
		_ = database.DisconnectPostgres()
	}()

	// Initialize container
	container, err := bootstrap.NewServiceContainer(rt.Config, dbClient, rt.Logger.Named("container"))
	if err != nil {
		return fmt.Errorf("container initialization failed: %w", err)
	}

	// Initialize HTTP server
	httpServer := http.NewServer(container, rt.Logger.Named("http"))

	// Initialize job scheduler
	scheduler, err := jobscheduler.NewScheduler(rt.Logger.Named("scheduler"), container, jobscheduler.SchedulerConfig{
		StartBackfillImmediately:  false,
		StartTemplateImmediately:  false,
		StartPriceSyncImmediately: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	supervisor := NewSupervisor(
		rt.Logger.Named("supervisor"),
		NewWorker("http", func() error { httpServer.Start(); return nil }, httpServer.Shutdown),
		NewWorker("scheduler", scheduler.Start, scheduler.Shutdown),
	)

	return supervisor.Run(ctx)
}
