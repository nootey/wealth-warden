package queue

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type RiverDispatcher struct {
	client        *river.Client[pgx.Tx]
	dispatchCount metric.Int64Counter
}

func NewRiverDispatcher(client *river.Client[pgx.Tx], meter metric.Meter) (*RiverDispatcher, error) {
	dispatchCount, err := meter.Int64Counter(
		"queue_job_dispatch_total",
		metric.WithDescription("Total number of jobs enqueued to the durable queue"),
	)
	if err != nil {
		return nil, err
	}
	return &RiverDispatcher{client: client, dispatchCount: dispatchCount}, nil
}

func (d *RiverDispatcher) Dispatch(ctx context.Context, job Job) error {
	metadata, err := InjectTraceContext(ctx)
	if err != nil {
		return fmt.Errorf("marshal trace context for job %s: %w", job.Kind(), err)
	}

	if _, err := d.client.Insert(ctx, job, &river.InsertOpts{Metadata: metadata}); err != nil {
		return fmt.Errorf("enqueue job %s: %w", job.Kind(), err)
	}

	d.dispatchCount.Add(ctx, 1, metric.WithAttributes(attribute.String("job_type", job.Kind())))
	return nil
}
