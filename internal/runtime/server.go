package runtime

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/http"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
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

	// Connect to DB
	dbClient, err := database.ConnectToPostgres(rt.Config)
	if err != nil {
		rt.Logger.Error("Database connection failed", zap.Error(err))
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() {
		err := database.DisconnectPostgres()
		if err != nil {
			fmt.Println("failed to disconnect postgres cleanly")
		}
	}()
	rt.Logger.Info("Successfully connected to the database")

	// Initialize container
	httpLogger := rt.Logger.Named("http").With(zap.String("component", "HTTP"))
	container, err := bootstrap.NewContainer(rt.Config, dbClient, httpLogger)
	if err != nil {
		rt.Logger.Error("Container initialization failed", zap.Error(err))
		return fmt.Errorf("cntainer initialization failed: %w", err)
	}

	// Start scheduler
	scheduler, err := rt.startScheduler(container)
	if err != nil {
		rt.Logger.Error("Failed to start scheduler", zap.Error(err))
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	defer func() {
		rt.Logger.Info("Scheduler exiting")
		if err := scheduler.Shutdown(); err != nil {
			rt.Logger.Error("Scheduler shutdown failed", zap.Error(err))
		}
	}()

	// Start HTTP server
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

func (rt *ServerRuntime) startScheduler(container *bootstrap.Container) (gocron.Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	// Schedule backfill job
	_, err = s.NewJob(
		gocron.DurationJob(12*time.Hour),
		gocron.NewTask(func() {
			rt.Logger.Info("Starting scheduled backfill job...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if err := rt.RunBackfill(ctx, container); err != nil {
				rt.Logger.Error("Backfill failed", zap.Error(err))
			} else {
				rt.Logger.Info("Backfill completed successfully")
			}
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return nil, err
	}

	s.Start()
	rt.Logger.Info("Scheduler started")
	return s, nil
}

// RunBackfill - Scheduled job to run balance backfills ... This is currently not production optimized for high throughput ... And it doesn't need to be ... For now
func (rt *ServerRuntime) RunBackfill(ctx context.Context, container *bootstrap.Container) error {

	// Get all active user IDs
	userIDs, err := container.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		rt.Logger.Info("No users to backfill")
		return nil
	}

	rt.Logger.Info("Backfilling balances", zap.Int("userCount", len(userIDs)))

	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	successCount := 0
	failCount := 0

	for _, userID := range userIDs {
		if err := container.AccountService.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
			rt.Logger.Error("Backfill failed for user",
				zap.Int64("userID", userID),
				zap.Error(err))
			failCount++
		} else {
			rt.Logger.Debug("Backfill completed for user", zap.Int64("userID", userID))
			successCount++
		}
	}

	rt.Logger.Info("Backfill completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	return nil
}
