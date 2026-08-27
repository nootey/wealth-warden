package queue_jobs

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
)

type NotificationArgs struct {
	Payload models.Notification
}

func (NotificationArgs) Kind() string { return TypeNotification }

type NotificationWorker struct {
	river.WorkerDefaults[NotificationArgs]
	repo        repositories.NotificationRepositoryInterface
	broadcaster ws.Broadcaster
}

func NewNotificationWorker(repo repositories.NotificationRepositoryInterface, broadcaster ws.Broadcaster) *NotificationWorker {
	return &NotificationWorker{repo: repo, broadcaster: broadcaster}
}

func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationArgs]) error {
	notification := job.Args.Payload
	if err := w.repo.Insert(ctx, &notification); err != nil {
		return err
	}
	w.broadcaster.Send(notification.UserID, ws.Event{Type: ws.TypeNotificationCreated})
	return nil
}
