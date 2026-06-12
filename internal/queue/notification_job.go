package queue

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type NotificationJob struct {
	Repo    repositories.NotificationRepositoryInterface `json:"-"`
	Payload models.Notification
}

func (j *NotificationJob) Type() string { return TypeNotification }

func (j *NotificationJob) Process(ctx context.Context) error {
	return j.Repo.Insert(ctx, &j.Payload)
}
