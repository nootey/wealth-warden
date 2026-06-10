package jobworker

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Consumer struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *registry

	workers           int
	pollInterval      time.Duration
	maxAttempts       int
	initialBackoff    time.Duration
	subsequentBackoff time.Duration
	visibilityTimeout time.Duration

	tracer      trace.Tracer
	jobDuration metric.Float64Histogram
	jobRuns     metric.Int64Counter
}

func NewConsumer(c *bootstrap.ServiceContainer, logger *zap.Logger, cfg config.QueueConfig) (*Consumer, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}

	cons := &Consumer{
		db:                c.DB,
		logger:            logger,
		registry:          newRegistry(c, logger),
		workers:           cfg.Workers,
		pollInterval:      time.Duration(cfg.PollIntervalMs) * time.Millisecond,
		maxAttempts:       cfg.MaxAttempts,
		initialBackoff:    time.Duration(cfg.RetryInitialBackoffSec) * time.Second,
		subsequentBackoff: time.Duration(cfg.RetrySubsequentBackoffSec) * time.Second,
		visibilityTimeout: time.Duration(cfg.VisibilityTimeoutSec) * time.Second,
		tracer:            otel.GetTracerProvider().Tracer(c.Config.Otel.ServiceName),
	}

	meter := otel.GetMeterProvider().Meter(c.Config.Otel.ServiceName)

	jobDuration, err := meter.Float64Histogram(
		"queue_job_duration_seconds",
		metric.WithDescription("Durable queue job processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	cons.jobDuration = jobDuration

	jobRuns, err := meter.Int64Counter(
		"queue_job_runs_total",
		metric.WithDescription("Total number of durable queue job executions"),
	)
	if err != nil {
		return nil, err
	}
	cons.jobRuns = jobRuns

	// Queue lag: age of the oldest pending job. Replaces the in-memory queue
	// depth gauge now that work lives in a table.
	_, err = meter.Int64ObservableGauge(
		"queue_oldest_pending_age_seconds",
		metric.WithDescription("Age in seconds of the oldest pending job"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			var ageSeconds int64
			err := cons.db.WithContext(ctx).
				Raw(`SELECT COALESCE(EXTRACT(EPOCH FROM now() - min(created_at)), 0)::bigint FROM jobs WHERE status = 'pending'`).
				Scan(&ageSeconds).Error
			if err != nil {
				return err
			}
			o.Observe(ageSeconds)
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	return cons, nil
}

func (c *Consumer) Name() string { return "job-consumer" }

func (c *Consumer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.worker(ctx)
		}()
	}
	wg.Wait()
}

func (c *Consumer) worker(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		claimed, err := c.processOne(ctx)
		if err != nil && ctx.Err() == nil {
			// Suppress errors caused by shutdown cancelling an in-flight claim.
			c.logger.Error("claim loop error", zap.Error(err))
		}
		if claimed {
			continue // keep draining while there's work
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(c.pollInterval):
		}
	}
}

// processOne atomically claims the next due job. Returns whether a job was claimed.
// It also reclaims jobs stuck in
// 'processing' past the visibility timeout (a worker that crashed mid-run), so
// jobs survive hard crashes.
func (c *Consumer) processOne(ctx context.Context) (bool, error) {
	var row models.Job
	res := c.db.WithContext(ctx).Raw(`
UPDATE jobs
SET status = 'processing', attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT id FROM jobs
    WHERE (status = 'pending' AND run_at <= now())
       OR (status = 'processing' AND updated_at < now() - make_interval(secs => ?))
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *`, int64(c.visibilityTimeout.Seconds())).Scan(&row)
	if res.Error != nil {
		return false, res.Error
	}
	if row.ID == 0 {
		return false, nil
	}

	job, err := c.registry.build(row.Type, row.Payload)
	if err != nil {
		// Unbuildable payload (unknown type / bad data) can never succeed — send
		// straight to the dead-letter rather than burning retries.
		c.logger.Error("failed to rebuild job, dead-lettering", zap.String("type", row.Type), zap.Int64("id", row.ID), zap.Error(err))
		c.markFailed(row, err)
		return true, nil
	}

	jobCtx := c.extractTraceContext(ctx, row.TraceCtx)
	if runErr := c.runJob(jobCtx, row, job); runErr != nil {
		c.handleFailure(row, runErr)
	} else {
		c.complete(row)
	}
	return true, nil
}

func (c *Consumer) runJob(ctx context.Context, row models.Job, job queue.Job) error {
	ctx, span := c.tracer.Start(ctx, "queue."+row.Type)
	defer span.End()

	start := time.Now()
	err := job.Process(ctx)
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	attrs := attribute.String("job_type", row.Type)
	c.jobDuration.Record(ctx, duration, metric.WithAttributes(attrs))
	c.jobRuns.Add(ctx, 1, metric.WithAttributes(attrs, attribute.String("status", status)))

	return err
}

func (c *Consumer) complete(row models.Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.db.WithContext(ctx).Delete(&models.Job{}, row.ID).Error; err != nil {
		c.logger.Error("failed to delete completed job", zap.Int64("id", row.ID), zap.Error(err))
	}
}

// retryBackoff decides the outcome of a failed attempt. row.Attempts already
// counts this run (the claim incremented it), so attempts == maxAttempts means
// the final try just failed and the job goes to the dead-letter.
func (c *Consumer) retryBackoff(attempts int) (backoff time.Duration, deadLetter bool) {
	if attempts >= c.maxAttempts {
		return 0, true
	}
	if attempts <= 1 {
		return c.initialBackoff, false
	}
	return c.subsequentBackoff, false
}

func (c *Consumer) handleFailure(row models.Job, runErr error) {
	backoff, deadLetter := c.retryBackoff(row.Attempts)
	if deadLetter {
		c.logger.Error("job dead-lettered", zap.String("type", row.Type), zap.Int64("id", row.ID), zap.Int("attempts", row.Attempts), zap.Error(runErr))
		c.markFailed(row, runErr)
		return
	}

	c.logger.Warn("job failed, scheduling retry", zap.String("type", row.Type), zap.Int64("id", row.ID), zap.Int("attempts", row.Attempts), zap.Duration("backoff", backoff), zap.Error(runErr))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"status":     models.JobStatusPending,
		"run_at":     time.Now().Add(backoff),
		"last_error": runErr.Error(),
	}).Error; err != nil {
		c.logger.Error("failed to reschedule job", zap.Int64("id", row.ID), zap.Error(err))
	}
}

func (c *Consumer) markFailed(row models.Job, runErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"status":     models.JobStatusFailed,
		"last_error": runErr.Error(),
	}).Error; err != nil {
		c.logger.Error("failed to mark job failed", zap.Int64("id", row.ID), zap.Error(err))
	}
}

func (c *Consumer) extractTraceContext(ctx context.Context, raw json.RawMessage) context.Context {
	if len(raw) == 0 {
		return ctx
	}
	carrier := propagation.MapCarrier{}
	if err := json.Unmarshal(raw, &carrier); err != nil {
		c.logger.Warn("failed to decode trace context", zap.Error(err))
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
