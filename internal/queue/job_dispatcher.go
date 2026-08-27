package queue

import (
	"context"

	"github.com/riverqueue/river"
)

// Job is River's job-args contract: data fields only. Kind() is persisted on
// river_job rows, so it must stay constant across struct renames.
type Job = river.JobArgs

type JobDispatcher interface {
	Dispatch(ctx context.Context, job Job) error
}
