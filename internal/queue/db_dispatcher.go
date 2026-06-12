package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"gorm.io/gorm"
)

// DBDispatcher persists jobs to the durable Postgres queue. Dispatch is a single
// INSERT, so it can run in the same transaction as a domain write when the caller passes a tx-scoped context.
type DBDispatcher struct {
	db            *gorm.DB
	dispatchCount metric.Int64Counter
}

func NewDBDispatcher(db *gorm.DB, meter metric.Meter) (*DBDispatcher, error) {
	dispatchCount, err := meter.Int64Counter(
		"queue_job_dispatch_total",
		metric.WithDescription("Total number of jobs enqueued to the durable queue"),
	)
	if err != nil {
		return nil, err
	}
	return &DBDispatcher{db: db, dispatchCount: dispatchCount}, nil
}

func (d *DBDispatcher) Dispatch(ctx context.Context, job Job) error {

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job %s: %w", job.Type(), err)
	}

	// Capture the live trace context so the consumer can nest the job span under the request/job that enqueued it.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	traceCtx, err := json.Marshal(carrier)
	if err != nil {
		return fmt.Errorf("marshal trace context for job %s: %w", job.Type(), err)
	}

	row := models.Job{
		Type:     job.Type(),
		Payload:  payload,
		Status:   models.JobStatusPending,
		RunAt:    time.Now(),
		TraceCtx: traceCtx,
	}
	if err := d.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("enqueue job %s: %w", job.Type(), err)
	}

	d.dispatchCount.Add(ctx, 1, metric.WithAttributes(attribute.String("job_type", job.Type())))
	return nil
}
