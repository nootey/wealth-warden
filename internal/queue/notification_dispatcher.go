package queue

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type NotificationDispatcher interface {
	Dispatch(userID int64, title, message string, notifType models.NotificationType) error
}

type notificationDispatcher struct {
	repo          repositories.NotificationRepositoryInterface
	jobDispatcher JobDispatcher
}

func NewNotificationDispatcher(repo repositories.NotificationRepositoryInterface, jobDispatcher JobDispatcher) NotificationDispatcher {
	return &notificationDispatcher{repo: repo, jobDispatcher: jobDispatcher}
}

func (d *notificationDispatcher) Dispatch(userID int64, title, message string, notifType models.NotificationType) error {
	return d.jobDispatcher.Dispatch(&NotificationJob{
		Repo: d.repo,
		Payload: models.Notification{
			UserID:  userID,
			Title:   title,
			Message: message,
			Type:    notifType,
		},
	})
}
