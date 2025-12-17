package runtime

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger    *zap.Logger
	container *bootstrap.Container
	scheduler gocron.Scheduler
}

func NewScheduler(logger *zap.Logger, container *bootstrap.Container) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		logger:    logger,
		container: container,
		scheduler: s,
	}, nil
}

func (s *Scheduler) Start() error {

	// Register jobs
	err := s.registerJobs()
	if err != nil {
		return err
	}

	s.scheduler.Start()
	s.logger.Info("Scheduler started")

	return nil
}

func (s *Scheduler) Shutdown() error {
	s.logger.Info("Scheduler shutting down")
	return s.scheduler.Shutdown()
}
func (s *Scheduler) registerJobs() error {

	err := s.registerBackfillJob()
	if err != nil {
		return err
	}

	err = s.registerTemplateJob()
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) registerBackfillJob() error {
	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
		gocron.NewTask(func() {
			s.logger.Info("Starting scheduled backfill job...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if err := s.runBackfill(ctx); err != nil {
				s.logger.Error("Backfill failed", zap.Error(err))
			} else {
				s.logger.Info("Backfill completed successfully")
			}
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	return err
}

func (s *Scheduler) registerTemplateJob() error {
	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 30, 0))),
		gocron.NewTask(func() {
			s.logger.Info("Starting scheduled template processing job...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := s.runTemplateProcessing(ctx); err != nil {
				s.logger.Error("Template processing failed", zap.Error(err))
			} else {
				s.logger.Info("Template processing completed successfully")
			}
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	return err
}

// RunBackfill - Scheduled job to run balance backfills ... This is currently not production optimized for high throughput ... And it doesn't need to be ... For now
func (s *Scheduler) runBackfill(ctx context.Context) error {

	// Get all active user IDs
	userIDs, err := s.container.UserService.GetAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		s.logger.Info("No users to backfill")
		return nil
	}

	s.logger.Info("Backfilling balances", zap.Int("userCount", len(userIDs)))

	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	successCount := 0
	failCount := 0

	for _, userID := range userIDs {
		if err := s.container.AccountService.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
			s.logger.Error("Backfill failed for user",
				zap.Int64("userID", userID),
				zap.Error(err))
			failCount++
		} else {
			s.logger.Debug("Backfill completed for user", zap.Int64("userID", userID))
			successCount++
		}
	}

	s.logger.Info("Backfill completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	return nil
}

func (s *Scheduler) runTemplateProcessing(ctx context.Context) error {

	templates, err := s.container.TransactionService.GetTemplatesReadyToRun(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		s.logger.Info("No templates ready to process")
		return nil
	}

	s.logger.Info("Processing templates", zap.Int("count", len(templates)))

	successCount := 0
	failCount := 0

	for _, template := range templates {
		if err := s.container.TransactionService.ProcessTemplate(ctx, template); err != nil {
			s.logger.Error("Failed to process template",
				zap.Int64("templateID", template.ID),
				zap.String("templateName", template.Name),
				zap.Error(err))
			failCount++
		} else {
			s.logger.Debug("Processed template",
				zap.Int64("templateID", template.ID),
				zap.String("templateName", template.Name))
			successCount++
		}
	}

	s.logger.Info("Template processing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	return nil
}
