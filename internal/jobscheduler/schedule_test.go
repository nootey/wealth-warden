package jobscheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
