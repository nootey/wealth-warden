package queue_jobs

import (
	"context"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/riverqueue/river"
)

type ActivityLogArgs struct {
	Event       string
	Category    string
	Description *string
	Payload     *utils.Changes
	Causer      *int64
}

func (ActivityLogArgs) Kind() string { return TypeActivityLog }

type ActivityLogWorker struct {
	river.WorkerDefaults[ActivityLogArgs]
	loggingRepo repositories.LoggingRepositoryInterface
}

func NewActivityLogWorker(loggingRepo repositories.LoggingRepositoryInterface) *ActivityLogWorker {
	return &ActivityLogWorker{loggingRepo: loggingRepo}
}

func (w *ActivityLogWorker) Work(ctx context.Context, job *river.Job[ActivityLogArgs]) error {
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
