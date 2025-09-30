package jobs

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"go.uber.org/zap"
)

type ActivityLogJob struct {
	Logger      *zap.Logger
	LoggingRepo *repositories.LoggingRepository
	Event       string
	Category    string
	Description *string
	Payload     *utils.Changes
	Causer      *int64
}

func (j *ActivityLogJob) Process() {
	err := j.LoggingRepo.InsertActivityLog(nil, j.Event, j.Category, j.Description, j.Payload, j.Causer)
	if err != nil {
		j.Logger.Error("Error processing activity log", zap.Error(err))
	}
}
