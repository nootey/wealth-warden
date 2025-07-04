package jobs

import (
	"go.uber.org/zap"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type AccessLogJob struct {
	Logger      *zap.Logger
	LoggingRepo *repositories.LoggingRepository
	Event       string
	Status      string
	Description *string
	Payload     *utils.Changes
	Causer      *models.User
}

func (j *AccessLogJob) Process() {
	err := j.LoggingRepo.InsertAccessLog(nil, j.Status, j.Event, j.Description, j.Payload, j.Causer)
	if err != nil {
		j.Logger.Error("Error processing activity log", zap.Error(err))
	}
}
