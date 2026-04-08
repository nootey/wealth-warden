package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type NotificationRepositoryInterface interface {
	Insert(ctx context.Context, n *models.Notification) error
	FindByUser(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]models.Notification, int64, error)
	MarkAsRead(ctx context.Context, userID, notificationID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
}

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

var _ NotificationRepositoryInterface = (*NotificationRepository)(nil)

func (r *NotificationRepository) Insert(ctx context.Context, n *models.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepository) FindByUser(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]models.Notification, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ?", userID)
	if onlyUnread {
		q = q.Where("read_at IS NULL")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []models.Notification
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, userID, notificationID int64) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", notificationID, userID).
		Update("read_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}
