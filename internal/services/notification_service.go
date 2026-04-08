package services

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type NotificationServiceInterface interface {
	GetNotifications(ctx context.Context, userID int64, onlyUnread bool, p utils.PaginationParams) ([]models.Notification, *utils.Paginator, error)
	MarkAsRead(ctx context.Context, userID, notificationID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
}

type NotificationService struct {
	repo repositories.NotificationRepositoryInterface
}

func NewNotificationService(repo repositories.NotificationRepositoryInterface) *NotificationService {
	return &NotificationService{repo: repo}
}

var _ NotificationServiceInterface = (*NotificationService)(nil)

func (s *NotificationService) GetNotifications(ctx context.Context, userID int64, onlyUnread bool, p utils.PaginationParams) ([]models.Notification, *utils.Paginator, error) {
	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, total, err := s.repo.FindByUser(ctx, userID, onlyUnread, p.RowsPerPage, offset)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(total) {
		from = int(total)
	}
	to := offset + len(records)
	if to > int(total) {
		to = int(total)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(total),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userID, notificationID int64) error {
	return s.repo.MarkAsRead(ctx, userID, notificationID)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
