package queue_jobs

import (
	"context"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type ActivityLogJob struct {
	LoggingRepo repositories.LoggingRepositoryInterface `json:"-"`
	Event       string
	Category    string
	Description *string
	Payload     *utils.Changes
	Causer      *int64
}

func (j *ActivityLogJob) Type() string { return TypeActivityLog }

func (j *ActivityLogJob) Process(ctx context.Context) error {
	return j.LoggingRepo.InsertActivityLog(
		ctx,
		nil, // tx
		j.Event,
		j.Category,
		j.Description,
		j.Payload,
		j.Causer,
	)
}
