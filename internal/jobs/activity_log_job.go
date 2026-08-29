package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"

	"github.com/riverqueue/river"
)

type ActivityLogWorker struct {
	river.WorkerDefaults[jobqueue.ActivityLogArgs]
	loggingRepo repositories.LoggingRepositoryInterface
}

func NewActivityLogWorker(loggingRepo repositories.LoggingRepositoryInterface) *ActivityLogWorker {
	return &ActivityLogWorker{loggingRepo: loggingRepo}
}

func (w *ActivityLogWorker) Work(ctx context.Context, job *river.Job[jobqueue.ActivityLogArgs]) error {
	return w.loggingRepo.InsertActivityLog(
		ctx,
		nil, // tx
		job.Args.Event,
		job.Args.Category,
		job.Args.Description,
		job.Args.Payload,
		job.Args.Causer,
	)
}
