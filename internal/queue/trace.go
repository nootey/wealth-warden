package queue

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// River owns the rest of river_job.metadata, so only this key is ours.
type traceMetadata struct {
	TraceCtx propagation.MapCarrier `json:"trace_ctx"`
}

// Returns nil when there is no active trace to carry.
func InjectTraceContext(ctx context.Context) ([]byte, error) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil, nil
	}
	return json.Marshal(traceMetadata{TraceCtx: carrier})
}

func ExtractTraceContext(ctx context.Context, metadata []byte) (context.Context, error) {
	if len(metadata) == 0 {
		return ctx, nil
	}
	var meta traceMetadata
	if err := json.Unmarshal(metadata, &meta); err != nil {
		return ctx, err
	}
	if len(meta.TraceCtx) == 0 {
		return ctx, nil
	}
	return otel.GetTextMapPropagator().Extract(ctx, meta.TraceCtx), nil
}
