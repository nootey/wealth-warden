package queue_jobs

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
)

type NotificationDispatcher interface {
	Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error
}

type notificationDispatcher struct {
	repo          repositories.NotificationRepositoryInterface
	jobDispatcher queue.JobDispatcher
}

func NewNotificationDispatcher(repo repositories.NotificationRepositoryInterface, jobDispatcher queue.JobDispatcher) NotificationDispatcher {
	return &notificationDispatcher{repo: repo, jobDispatcher: jobDispatcher}
}

func (d *notificationDispatcher) Dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) error {
	return d.jobDispatcher.Dispatch(ctx, &NotificationJob{
		Repo: d.repo,
		Payload: models.Notification{
			UserID:  userID,
			Title:   title,
			Message: message,
			Type:    notifType,
		},
	})
}
