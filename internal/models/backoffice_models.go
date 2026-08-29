package models

import (
	"encoding/json"
	"time"
)

var RiverJobStates = []string{
	"available",
	"cancelled",
	"completed",
	"discarded",
	"pending",
	"retryable",
	"running",
	"scheduled",
}

var RiverJobSortFields = map[string]string{
	"id":           "id",
	"kind":         "kind",
	"queue":        "queue",
	"state":        "state",
	"attempt":      "attempt",
	"created_at":   "created_at",
	"scheduled_at": "scheduled_at",
	"attempted_at": "attempted_at",
	"finalized_at": "finalized_at",
	"duration":     "(COALESCE(finalized_at, now()) - attempted_at)",
}

type RiverJobRow struct {
	ID          int64           `gorm:"column:id" json:"id"`
	Kind        string          `gorm:"column:kind" json:"kind"`
	Queue       string          `gorm:"column:queue" json:"queue"`
	State       string          `gorm:"column:state" json:"state"`
	Attempt     int             `gorm:"column:attempt" json:"attempt"`
	MaxAttempts int             `gorm:"column:max_attempts" json:"max_attempts"`
	Priority    int             `gorm:"column:priority" json:"priority"`
	Args        json.RawMessage `gorm:"column:args" json:"args"`
	Metadata    json.RawMessage `gorm:"column:metadata" json:"metadata"`
	CreatedAt   time.Time       `gorm:"column:created_at" json:"created_at"`
	ScheduledAt time.Time       `gorm:"column:scheduled_at" json:"scheduled_at"`
	AttemptedAt *time.Time      `gorm:"column:attempted_at" json:"attempted_at"`
	FinalizedAt *time.Time      `gorm:"column:finalized_at" json:"finalized_at"`
}

func (RiverJobRow) TableName() string { return "river_job" }

type RiverJobStateCount struct {
	State string `gorm:"column:state" json:"state"`
	Count int64  `gorm:"column:count" json:"count"`
}

type RiverQueueStateCount struct {
	Queue string `gorm:"column:queue" json:"queue"`
	State string `gorm:"column:state" json:"state"`
	Count int64  `gorm:"column:count" json:"count"`
}

type RiverKindLastRun struct {
	Kind      string    `gorm:"column:kind" json:"kind"`
	LastRunAt time.Time `gorm:"column:last_run_at" json:"last_run_at"`
}

type RiverAttemptError struct {
	At      time.Time `json:"at"`
	Attempt int       `json:"attempt"`
	Error   string    `json:"error"`
	Trace   string    `json:"trace"`
}

type RiverJobDetail struct {
	ID          int64               `json:"id"`
	Kind        string              `json:"kind"`
	Queue       string              `json:"queue"`
	State       string              `json:"state"`
	Attempt     int                 `json:"attempt"`
	MaxAttempts int                 `json:"max_attempts"`
	Priority    int                 `json:"priority"`
	AttemptedBy []string            `json:"attempted_by"`
	Tags        []string            `json:"tags"`
	Args        json.RawMessage     `json:"args"`
	Metadata    json.RawMessage     `json:"metadata"`
	Errors      []RiverAttemptError `json:"errors"`
	CreatedAt   time.Time           `json:"created_at"`
	ScheduledAt time.Time           `json:"scheduled_at"`
	AttemptedAt *time.Time          `json:"attempted_at"`
	FinalizedAt *time.Time          `json:"finalized_at"`
}

type RiverQueueRow struct {
	Name     string           `json:"name"`
	PausedAt *time.Time       `json:"paused_at"`
	Counts   map[string]int64 `json:"counts"`
}

type RiverPeriodicJob struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Schedule  string     `json:"schedule"`
	Queue     string     `json:"queue"`
	LastRunAt *time.Time `json:"last_run_at"`
}

type ZeroCostMigrationResult struct {
	TotalProcessed  int `json:"total_processed"`
	AssetsProcessed int `json:"assets_processed"`
	AssetsFailed    int `json:"assets_failed"`
}
