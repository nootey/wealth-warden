package runtime

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/pkg/prices"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger    *zap.Logger
	container *bootstrap.Container
	scheduler gocron.Scheduler
	config    SchedulerConfig
}

type SchedulerConfig struct {
	StartBackfillImmediately  bool
	StartTemplateImmediately  bool
	StartPriceSyncImmediately bool
}

func NewScheduler(logger *zap.Logger, container *bootstrap.Container, config SchedulerConfig) (*Scheduler, error) {

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if container == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		logger:    logger,
		container: container,
		scheduler: s,
		config:    config,
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

	err = s.registerInvestmentPriceSyncJob()
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) registerBackfillJob() error {

	job := jobscheduler.NewBackfillJob(s.logger, s.container)

	var opts []gocron.JobOption
	if s.config.StartBackfillImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
		gocron.NewTask(func() {
			s.logger.Info("Starting scheduled backfill job...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				s.logger.Error("Backfill failed", zap.Error(err))
			} else {
				s.logger.Info("Backfill completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerTemplateJob() error {

	job := jobscheduler.NewAutomateTemplateJob(s.logger, s.container)

	var opts []gocron.JobOption
	if s.config.StartTemplateImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 30, 0))),
		gocron.NewTask(func() {
			s.logger.Info("Starting scheduled template processing job...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				s.logger.Error("Template processing failed", zap.Error(err))
			} else {
				s.logger.Info("Template processing completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerInvestmentPriceSyncJob() error {

	// Create price fetch client
	client, err := prices.NewPriceFetchClient(s.container.Config.FinanceAPIBaseURL)
	if err != nil {
		s.logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	job := jobscheduler.NewInvestmentPriceSyncJob(s.logger, s.container, client)

	var opts []gocron.JobOption
	if s.config.StartPriceSyncImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err = s.scheduler.NewJob(
		gocron.DurationJob(12*time.Hour),
		gocron.NewTask(func() {
			s.logger.Info("Starting investment price sync ...")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				s.logger.Error("Price sync failed", zap.Error(err))
			} else {
				s.logger.Info("Price sync completed")
			}
		}),
		opts...,
	)
	return err
}
