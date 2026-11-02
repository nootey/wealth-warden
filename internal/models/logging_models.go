package models

import (
	"time"

	"gorm.io/datatypes"
)

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

var ActivityLogSort = SortSpec[ActivityLog]{
	Fields: map[string]SortKey[ActivityLog]{
		"created_at": {Column: "activity_logs.created_at", Value: func(l ActivityLog) any { return l.CreatedAt }},
		"event":      {Column: "activity_logs.event", Value: func(l ActivityLog) any { return l.Event }},
		"category":   {Column: "activity_logs.category", Value: func(l ActivityLog) any { return l.Category }},
		"causer_id":  {Column: "activity_logs.causer_id", Value: func(l ActivityLog) any { return l.CauserID }},
	},
	ID:      SortKey[ActivityLog]{Column: "activity_logs.id", Value: func(l ActivityLog) any { return l.ID }},
	Default: "created_at",
}
