package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
)

type NotificationWorker struct {
	river.WorkerDefaults[jobqueue.NotificationArgs]
	repo        repositories.NotificationRepositoryInterface
	broadcaster ws.Broadcaster
}

func NewNotificationWorker(repo repositories.NotificationRepositoryInterface, broadcaster ws.Broadcaster) *NotificationWorker {
	return &NotificationWorker{repo: repo, broadcaster: broadcaster}
}

func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[jobqueue.NotificationArgs]) error {
	notification := job.Args.Payload
	if err := w.repo.Insert(ctx, &notification); err != nil {
		return err
	}
	w.broadcaster.Send(notification.UserID, ws.Event{Type: ws.TypeNotificationCreated})
	return nil
}
