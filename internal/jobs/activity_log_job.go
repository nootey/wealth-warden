package jobs

import (
	"context"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type ActivityLogJob struct {
	LoggingRepo repositories.LoggingRepositoryInterface
	Event       string
	Category    string
	Description *string
	Payload     *utils.Changes
	Causer      *int64
}

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
