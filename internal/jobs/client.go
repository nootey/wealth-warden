package jobs

import (
	"context"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	riverStopTimeout = 30 * time.Second

	defaultPollInterval = time.Second
	defaultJobTimeout   = 15 * time.Minute

	schedulerQueueWorkers = 2

	// Retention windows for finished jobs. River's defaults are 24h, the backoffice job monitor needs them to be longer.
	completedJobRetention = 7 * 24 * time.Hour
	cancelledJobRetention = 7 * 24 * time.Hour
	discardedJobRetention = 30 * 24 * time.Hour
)

func NewClient(pool *pgxpool.Pool, logger *zap.Logger, cfg config.QueueConfig, serviceName string, periodicJobs []*river.PeriodicJob) (*river.Client[pgx.Tx], *river.Workers, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}

	pollInterval := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	jobTimeout := time.Duration(cfg.JobTimeoutSec) * time.Second
	if jobTimeout <= 0 {
		jobTimeout = defaultJobTimeout
	}

	meter := otel.GetMeterProvider().Meter(serviceName)
	metrics, err := newMetricsMiddleware(meter)
	if err != nil {
		return nil, nil, err
	}

	workers := river.NewWorkers()

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:      {MaxWorkers: cfg.Workers},
			jobqueue.QueueScheduler: {MaxWorkers: schedulerQueueWorkers},
			// One worker is what serialises the investment rebuilds.
			jobqueue.QueueRebuild: {MaxWorkers: 1},
		},
		PeriodicJobs:      periodicJobs,
		MaxAttempts:       cfg.MaxAttempts,
		FetchPollInterval: pollInterval,
		JobTimeout:        jobTimeout,

		CompletedJobRetentionPeriod: completedJobRetention,
		CancelledJobRetentionPeriod: cancelledJobRetention,
		DiscardedJobRetentionPeriod: discardedJobRetention,

		Middleware: []rivertype.Middleware{
			&traceMiddleware{logger: logger},
			otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
				MeterProvider:  otel.GetMeterProvider(),
				TracerProvider: otel.GetTracerProvider(),
			}),
			metrics,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return client, workers, nil
}

type traceMiddleware struct {
	river.MiddlewareDefaults
	logger *zap.Logger
}

func (m *traceMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	jobCtx, err := jobqueue.ExtractTraceContext(ctx, job.Metadata)
	if err != nil {
		m.logger.Warn("failed to decode trace context", zap.String("kind", job.Kind), zap.Error(err))
	}
	return doInner(jobCtx)
}

type metricsMiddleware struct {
	river.MiddlewareDefaults
	jobDuration metric.Float64Histogram
	jobRuns     metric.Int64Counter
}

func newMetricsMiddleware(meter metric.Meter) (*metricsMiddleware, error) {
	jobDuration, err := meter.Float64Histogram(
		"queue_job_duration_seconds",
		metric.WithDescription("Durable queue job processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	jobRuns, err := meter.Int64Counter(
		"queue_job_runs_total",
		metric.WithDescription("Total number of durable queue job executions"),
	)
	if err != nil {
		return nil, err
	}

	return &metricsMiddleware{jobDuration: jobDuration, jobRuns: jobRuns}, nil
}

func (m *metricsMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	start := time.Now()
	err := doInner(ctx)
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
	}

	// The queue label is what splits the scheduler and async panels in Grafana.
	attrs := []attribute.KeyValue{
		attribute.String("job_type", job.Kind),
		attribute.String("queue", job.Queue),
	}
	m.jobDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
	m.jobRuns.Add(ctx, 1, metric.WithAttributes(append(attrs, attribute.String("status", status))...))

	return err
}

type Service struct {
	client *river.Client[pgx.Tx]
	logger *zap.Logger
}

func NewService(client *river.Client[pgx.Tx], logger *zap.Logger) *Service {
	return &Service{client: client, logger: logger}
}

func (s *Service) Run(ctx context.Context) {
	if err := s.client.Start(ctx); err != nil {
		s.logger.Error("river client failed to start", zap.Error(err))
		return
	}

	<-ctx.Done()

	// The run context is already cancelled; in-flight jobs need their own deadline.
	stopCtx, cancel := context.WithTimeout(context.Background(), riverStopTimeout)
	defer cancel()
	if err := s.client.Stop(stopCtx); err != nil {
		s.logger.Error("river client shutdown failed", zap.Error(err))
	}
}

func RegisterQueueLagGauge(repo repositories.JobRepositoryInterface, meter metric.Meter) error {
	_, err := meter.Int64ObservableGauge(
		"queue_oldest_pending_age_seconds",
		metric.WithDescription("Age in seconds of the oldest pending job"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			ageSeconds, err := repo.OldestPendingAgeSeconds(ctx)
			if err != nil {
				return err
			}
			o.Observe(ageSeconds)
			return nil
		}),
	)
	return err
}
