package models

import (
	"gorm.io/datatypes"
	"time"
)

type AccessLog struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Event       string         `gorm:"type:varchar(255);not null" json:"event"`
	Service     *string        `gorm:"type:varchar(255)" json:"service,omitempty"`
	Status      string         `gorm:"type:varchar(255);not null" json:"status"`
	IPAddress   *string        `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent   *string        `gorm:"type:text" json:"user_agent,omitempty"`
	CauserID    *uint          `gorm:"index" json:"causer_id,omitempty"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

type ActivityLog struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Event       string         `gorm:"type:varchar(255);not null" json:"event"`
	Category    string         `gorm:"type:varchar(255);not null" json:"category"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	CauserID    *uint          `gorm:"index" json:"causer_id,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

type NotificationLog struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      *uint          `gorm:"index" json:"user_id,omitempty"`
	Type        string         `gorm:"type:varchar(255);not null" json:"type"`
	Destination *string        `gorm:"type:varchar(255)" json:"destination,omitempty"`
	Status      string         `gorm:"type:varchar(50);not null" json:"status"`
	Message     *string        `gorm:"type:text" json:"message,omitempty"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}
