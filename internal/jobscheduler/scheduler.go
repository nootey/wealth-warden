package jobscheduler

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"

	"github.com/go-co-op/gocron/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	jobNameAssetHistoryBackfill = "asset-history-backfill-job"
	jobNameBalanceBackfill      = "balance-backfill-job"
	jobNameTemplates            = "templates-job"
	jobNameSavingsGoalFund      = "savings-goal-fund-job"
	jobNameAssetPriceSync       = "asset-price-sync-job"
)

type Scheduler struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	scheduler         gocron.Scheduler
	flags             SchedulerFlags
	concurrentWorkers int
	tracer            trace.Tracer
	jobDuration       metric.Float64Histogram
	jobRuns           metric.Int64Counter
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

	meter := otel.GetMeterProvider().Meter("wealth-warden")

	jobDuration, err := meter.Float64Histogram(
		"scheduler_job_duration_seconds",
		metric.WithDescription("Scheduler job execution duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	jobRuns, err := meter.Int64Counter(
		"scheduler_job_runs_total",
		metric.WithDescription("Total number of scheduler job executions"),
	)
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		logger:            logger,
		container:         container,
		scheduler:         s,
		flags:             flags,
		concurrentWorkers: concurrentWorkers,
		tracer:            otel.GetTracerProvider().Tracer("wealth-warden"),
		jobDuration:       jobDuration,
		jobRuns:           jobRuns,
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

func (s *Scheduler) runJob(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := s.tracer.Start(ctx, "scheduler."+name)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	attrs := attribute.String("job_name", name)
	s.jobDuration.Record(ctx, duration, metric.WithAttributes(attrs))
	s.jobRuns.Add(ctx, 1, metric.WithAttributes(attrs, attribute.String("status", status)))

	return err
}

func (s *Scheduler) registerJobs() error {

	err := s.registerAssetPriceHistoryBackfillJob()
	if err != nil {
		return err
	}

	err = s.registerBackfillJob()
	if err != nil {
		return err
	}

	err = s.registerTemplateAndFundSavingsJobs()
	if err != nil {
		return err
	}

	err = s.registerAssetPriceSyncJob()
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) registerAssetPriceHistoryBackfillJob() error {

	logger := s.logger.Named(jobNameAssetHistoryBackfill)
	job := NewAssetPriceHistoryBackfillJob(logger, s.container.InvestmentService, s.container.DB)

	var opts []gocron.JobOption
	if s.flags.StartAssetHistoryBackfillImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting asset price history backfill ...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := s.runJob(ctx, jobNameAssetHistoryBackfill, job.Run); err != nil {
				logger.Error("Price history backfill failed", zap.Error(err))
			} else {
				logger.Info("Price history backfill completed")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerBackfillJob() error {

	logger := s.logger.Named(jobNameBalanceBackfill)
	job := NewBalanceBackfillJob(logger, s.container, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartBalanceBackfillImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 6, 0))),
		gocron.NewTask(func() {
			logger.Info("Starting scheduled backfill job...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if err := s.runJob(ctx, jobNameBalanceBackfill, job.Run); err != nil {
				logger.Error("Backfill failed", zap.Error(err))
			} else {
				logger.Info("Backfill completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerTemplateAndFundSavingsJobs() error {

	tLogger := s.logger.Named(jobNameTemplates)
	templateJob := NewAutomateTemplateJob(tLogger, s.container, s.container.NotifDispatcher, s.concurrentWorkers)

	sLogger := s.logger.Named(jobNameSavingsGoalFund)
	savingsJob := NewAutoFundGoalsJob(sLogger, s.container, s.container.NotifDispatcher, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartTemplatesImmediately || s.flags.StartSavingsGoalFundImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	_, err := s.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 10, 0))),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()

			tLogger.Info("Starting scheduled template processing job...")
			if err := s.runJob(ctx, jobNameTemplates, templateJob.Run); err != nil {
				tLogger.Error("Template processing failed", zap.Error(err))
			} else {
				tLogger.Info("Template processing completed successfully")
			}

			sLogger.Info("Starting savings goal auto-fund job...")
			if err := s.runJob(ctx, jobNameSavingsGoalFund, savingsJob.Run); err != nil {
				sLogger.Error("Savings goal auto-fund failed", zap.Error(err))
			} else {
				sLogger.Info("Savings goal auto-fund completed successfully")
			}
		}),
		opts...,
	)
	return err
}

func (s *Scheduler) registerAssetPriceSyncJob() error {

	logger := s.logger.Named(jobNameAssetPriceSync)
	client, err := finance.NewPriceFetchClient(s.container.Config.FinanceAPIBaseURL)
	if err != nil {
		logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	job := NewAssetPriceSyncJob(logger, s.container.InvestmentService, s.container.AccountService, s.container.DB, client, s.container.NotifDispatcher, s.concurrentWorkers)

	var opts []gocron.JobOption
	if s.flags.StartAssetPriceSyncImmediately {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	opts = append(opts, gocron.WithSingletonMode(gocron.LimitModeReschedule))

	_, err = s.scheduler.NewJob(
		gocron.DurationJob(8*time.Hour),
		gocron.NewTask(func() {
			logger.Info("Starting asset price sync ...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := s.runJob(ctx, jobNameAssetPriceSync, job.Run); err != nil {
				logger.Error("Price sync failed", zap.Error(err))
			} else {
				logger.Info("Price sync completed")
			}
		}),
		opts...,
	)
	return err
}
