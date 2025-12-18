package runtime

import (
	"context"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobscheduler"

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

	job := jobscheduler.NewBackfillJob(s.logger, s.container)

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
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	return err
}

func (s *Scheduler) registerTemplateJob() error {

	job := jobscheduler.NewAutomateTemplateJob(s.logger, s.container)

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
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	return err
}
