package jobqueue

import (
	"context"
	"wealth-warden/internal/models"
)

type NotificationDispatcher interface {
	Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error
}

type notificationDispatcher struct {
	jobDispatcher Dispatcher
}

func NewNotificationDispatcher(jobDispatcher Dispatcher) NotificationDispatcher {
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
