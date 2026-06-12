package queue

import "context"

type Job interface {
	Process(ctx context.Context) error
	// Type returns the stable registry key for this job. It is persisted on the
	// jobs row and reused as the metric/trace job_type label, so it must stay
	// constant across struct renames.
	Type() string
}

type JobDispatcher interface {
	Dispatch(ctx context.Context, job Job) error
}
