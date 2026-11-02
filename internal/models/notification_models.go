package models

import "time"

type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

type Notification struct {
	ID        int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64            `gorm:"not null;index" json:"user_id"`
	Title     string           `gorm:"type:varchar(255);not null" json:"title"`
	Message   string           `gorm:"type:text;not null" json:"message"`
	Type      NotificationType `gorm:"type:notification_type;not null;default:info" json:"type"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

var NotificationSort = SortSpec[Notification]{
	Fields: map[string]SortKey[Notification]{
		"created_at": {Column: "notifications.created_at", Value: func(n Notification) any { return n.CreatedAt }},
	},
	ID:      SortKey[Notification]{Column: "notifications.id", Value: func(n Notification) any { return n.ID }},
	Default: "created_at",
}
