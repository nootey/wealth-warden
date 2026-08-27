package jobs

import (
	"testing"
	"time"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/require"
)

// A typo on either side of the config string is silent: the job just never
// starts on boot.
func TestPeriodicSpecs_ImmediateJobsMapToRunOnStart(t *testing.T) {
	all := []string{
		"asset_history_backfill",
		"balance_backfill",
		"recurring_transactions",
		"asset_price_sync",
	}

	none := periodicSpecs(config.SchedulerConfig{})
	require.Len(t, none, len(all))
	for _, spec := range none {
		require.False(t, spec.runOnStart, "%s should not run on start by default", spec.args.Kind())
	}

	every := periodicSpecs(config.SchedulerConfig{ImmediateJobs: all})
	require.Len(t, every, len(all))
	for _, spec := range every {
		require.True(t, spec.runOnStart, "%s should run on start", spec.args.Kind())
	}
}

func TestPeriodicSpecs_KindsAreUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, spec := range periodicSpecs(config.SchedulerConfig{}) {
		kind := spec.args.Kind()
		require.False(t, seen[kind], "duplicate kind %q", kind)
		seen[kind] = true
	}
}

func TestDailyAt_Next(t *testing.T) {
	schedule := dailyAt{hour: 0, minute: 10}

	tests := []struct {
		name    string
		current time.Time
		want    time.Time
	}{
		{
			name:    "before the target time, same day",
			current: time.Date(2026, 8, 25, 0, 5, 0, 0, time.Local),
			want:    time.Date(2026, 8, 25, 0, 10, 0, 0, time.Local),
		},
		{
			name:    "after the target time, next day",
			current: time.Date(2026, 8, 25, 13, 0, 0, 0, time.Local),
			want:    time.Date(2026, 8, 26, 0, 10, 0, 0, time.Local),
		},
		{
			name:    "exactly at the target time, next day",
			current: time.Date(2026, 8, 25, 0, 10, 0, 0, time.Local),
			want:    time.Date(2026, 8, 26, 0, 10, 0, 0, time.Local),
		},
		{
			name:    "rolls over a month boundary",
			current: time.Date(2026, 8, 31, 23, 0, 0, 0, time.Local),
			want:    time.Date(2026, 9, 1, 0, 10, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, schedule.Next(tt.current))
		})
	}
}
