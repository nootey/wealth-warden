package queue_jobs

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/ws"
)

type NotificationJob struct {
	Repo        repositories.NotificationRepositoryInterface `json:"-"`
	Broadcaster ws.Broadcaster                               `json:"-"`
	Payload     models.Notification
}

func (j *NotificationJob) Type() string { return TypeNotification }

func (j *NotificationJob) Process(ctx context.Context) error {
	if err := j.Repo.Insert(ctx, &j.Payload); err != nil {
		return err
	}
	j.Broadcaster.Send(j.Payload.UserID, ws.Event{Type: ws.TypeNotificationCreated})
	return nil
}
