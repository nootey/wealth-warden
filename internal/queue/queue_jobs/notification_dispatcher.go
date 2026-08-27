package queue_jobs

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
)

type NotificationDispatcher interface {
	Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error
}

type notificationDispatcher struct {
	jobDispatcher queue.JobDispatcher
}

func NewNotificationDispatcher(jobDispatcher queue.JobDispatcher) NotificationDispatcher {
	return &notificationDispatcher{jobDispatcher: jobDispatcher}
}

func (d *notificationDispatcher) Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error {
	return d.jobDispatcher.Dispatch(ctx, NotificationArgs{
		Payload: models.Notification{
			UserID:  userID,
			Title:   title,
			Message: message,
			Type:    notifType,
		},
	})
}
