package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
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

var RiverJobSort = SortSpec[RiverJobRow]{
	Fields: map[string]SortKey[RiverJobRow]{
		"kind":         {Column: "river_job.kind", Value: func(j RiverJobRow) any { return j.Kind }},
		"queue":        {Column: "river_job.queue", Value: func(j RiverJobRow) any { return j.Queue }},
		"state":        {Column: "river_job.state", Value: func(j RiverJobRow) any { return j.State }},
		"attempt":      {Column: "river_job.attempt", Value: func(j RiverJobRow) any { return j.Attempt }},
		"created_at":   {Column: "river_job.created_at", Value: func(j RiverJobRow) any { return j.CreatedAt }},
		"scheduled_at": {Column: "river_job.scheduled_at", Value: func(j RiverJobRow) any { return j.ScheduledAt }},
		"attempted_at": {Column: "river_job.attempted_at", Value: func(j RiverJobRow) any { return j.AttemptedAt }},
		"finalized_at": {Column: "river_job.finalized_at", Value: func(j RiverJobRow) any { return j.FinalizedAt }},
		// A running job's duration grows between requests, so paging by it can skip or repeat
		// a running job. Finished jobs compare exactly to the microsecond.
		"duration": {
			Column: "EXTRACT(EPOCH FROM (COALESCE(river_job.finalized_at, now()) - river_job.attempted_at))",
			Value: func(j RiverJobRow) any {
				if j.AttemptedAt == nil {
					return nil
				}
				end := time.Now()
				if j.FinalizedAt != nil {
					end = *j.FinalizedAt
				}
				return decimal.New(end.Sub(*j.AttemptedAt).Microseconds(), -6)
			},
		},
	},
	ID:      SortKey[RiverJobRow]{Column: "river_job.id", Value: func(j RiverJobRow) any { return j.ID }},
	Default: "id",
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
