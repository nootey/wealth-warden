package models

import (
	"gorm.io/datatypes"
	"time"
)

type AccessLog struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Status      string         `gorm:"type:varchar(255);not null" json:"status"`
	Event       string         `gorm:"type:varchar(255);not null" json:"event"`
	CauserID    *int64         `gorm:"index" json:"causer_id,omitempty"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

type ActivityLog struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Event       string         `gorm:"type:varchar(255);not null" json:"event"`
	Category    string         `gorm:"type:varchar(255);not null" json:"category"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	CauserID    *int64         `gorm:"index" json:"causer_id,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}
