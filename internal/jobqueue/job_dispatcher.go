package jobqueue

import "context"

type Job interface {
	Process(ctx context.Context) error
}

type JobDispatcher interface {
	Dispatch(job Job) error
}
