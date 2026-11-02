package services

import (
	"context"
	"errors"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
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
	p.SortField, p.SortOrder = "created_at", "desc"
	ks, err := utils.NewKeyset(models.NotificationSort, p)
	if err != nil {
		return nil, nil, err
	}

	records, total, err := s.repo.FindByUser(ctx, userID, onlyUnread, ks)
	if err != nil {
		return nil, nil, utils.KeysetError(ks, err)
	}

	records, paginator := utils.Paginate(records, ks, models.NotificationSort, total)
	return records, paginator, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userID, notificationID int64) error {
	err := s.repo.MarkAsRead(ctx, userID, notificationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.Wrap(apperr.NotFound, "Notification not found", err)
	}
	return err
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
