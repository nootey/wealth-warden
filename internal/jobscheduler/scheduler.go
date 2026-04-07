package jobscheduler

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	scheduler         gocron.Scheduler
	flags             SchedulerFlags
	concurrentWorkers int
}

type SchedulerFlags struct {
	StartBalanceBackfillImmediately      bool
	StartTemplatesImmediately            bool
	StartAssetPriceSyncImmediately       bool
	StartAssetHistoryBackfillImmediately bool
	StartSavingsGoalFundImmediately      bool
}

func FlagsFromConfig(cfg config.SchedulerConfig) SchedulerFlags {
	flags := SchedulerFlags{}
	for _, job := range cfg.ImmediateJobs {
		switch job {
		case "balance_backfill":
			flags.StartBalanceBackfillImmediately = true
		case "templates":
			flags.StartTemplatesImmediately = true
		case "asset_price_sync":
			flags.StartAssetPriceSyncImmediately = true
		case "asset_history_backfill":
			flags.StartAssetHistoryBackfillImmediately = true
		case "savings_goal_fund":
			flags.StartSavingsGoalFundImmediately = true
		}
	}
	return flags
}

func NewScheduler(logger *zap.Logger, container *bootstrap.ServiceContainer, flags SchedulerFlags, concurrentWorkers int) (*Scheduler, error) {

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

	if concurrentWorkers <= 0 {
		concurrentWorkers = 5
	}

	return &Scheduler{
		logger:            logger,
		container:         container,
		scheduler:         s,
		flags:             flags,
		concurrentWorkers: concurrentWorkers,
	}, nil
}

func (s *Scheduler) Name() string { return "scheduler" }

func (s *Scheduler) Start(ctx context.Context) error {

	// Register jobs
	err := s.registerJobs()
	if err != nil {
		return err
	}

	s.scheduler.Start()
	s.logger.Info("Scheduler started")

	<-ctx.Done()
	return nil
}

func (s *Scheduler) Shutdown() error {
	s.logger.Info("Scheduler shutting down")
	return s.scheduler.Shutdown()
}

func (s *Scheduler) registerJobs() error {

	err := s.registerAssetPriceSyncJob()
	if err != nil {
		return err
	}

	err = s.registerBackfillJob()
	if err != nil {
		return err
	}

	err = s.registerTemplatesJob()
	if err != nil {
		return err
	}

	err = s.registerSavingsGoalFundJob()
	if err != nil {
		return err
	}

	err = s.registerAssetPriceHistoryBackfillJob()
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) registerBackfillJob() error {

	logger := s.logger.Named("balance-backfill-job")
	job := NewBalanceBackfillJob(logger, s.container, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartBalanceBackfillImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 5, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting scheduled backfill job...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				logger.Error("Backfill failed", zap.Error(err))
			} else {
				logger.Info("Backfill completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerTemplatesJob() error {

	logger := s.logger.Named("templates-job")
	job := NewAutomateTemplateJob(logger, s.container, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartTemplatesImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 30, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting scheduled template processing job...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				logger.Error("Template processing failed", zap.Error(err))
			} else {
				logger.Info("Template processing completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerAssetPriceSyncJob() error {

	logger := s.logger.Named("asset-price-sync-job")
	client, err := finance.NewPriceFetchClient(s.container.Config.FinanceAPIBaseURL)
	if err != nil {
		logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	job := NewAssetPriceSyncJob(logger, s.container.InvestmentService, s.container.AccountService, s.container.DB, client, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartAssetPriceSyncImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	opts = append(opts, gocron.WithSingletonMode(gocron.LimitModeReschedule))

	_, err = s.scheduler.NewJob(
		gocron.DurationJob(8*time.Hour),
		gocron.NewTask(func() {
			logger.Info("Starting asset price sync ...")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				logger.Error("Price sync failed", zap.Error(err))
			} else {
				logger.Info("Price sync completed")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerSavingsGoalFundJob() error {

	logger := s.logger.Named("savings-goal-fund-job")
	job := NewAutoFundGoalsJob(logger, s.container, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartSavingsGoalFundImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 40, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting savings goal auto-fund job...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				logger.Error("Savings goal auto-fund failed", zap.Error(err))
			} else {
				logger.Info("Savings goal auto-fund completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerAssetPriceHistoryBackfillJob() error {

	logger := s.logger.Named("asset-history-backfill-job")
	job := NewAssetPriceHistoryBackfillJob(logger, s.container.InvestmentService, s.container.DB)

	var opts []gocron.JobOption
	if s.flags.StartAssetHistoryBackfillImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting asset price history backfill ...")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if err := job.Run(ctx); err != nil {
				logger.Error("Price history backfill failed", zap.Error(err))
			} else {
				logger.Info("Price history backfill completed")
			}
		}),
		opts...,
	)
	return err
}
