package models

import (
	"encoding/json"
	"time"
)

// There is deliberately no "completed" state: successful jobs are deleted by the consumer. "failed" rows are the dead-letter.
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusFailed     = "failed"
)

type Job struct {
	ID        int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string          `gorm:"size:64;not null" json:"type"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null" json:"payload"`
	Status    string          `gorm:"size:16;not null;default:'pending'" json:"status"`
	Attempts  int             `gorm:"not null;default:0" json:"attempts"`
	RunAt     time.Time       `gorm:"not null" json:"run_at"`
	LastError *string         `gorm:"type:text" json:"last_error,omitempty"`
	TraceCtx  json.RawMessage `gorm:"type:jsonb" json:"trace_ctx,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (Job) TableName() string { return "jobs" }
