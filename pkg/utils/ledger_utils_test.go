package utils_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestValidateAccount(t *testing.T) {
	t.Run("valid account", func(t *testing.T) {
		acc := &models.Account{ID: 1, IsActive: true, ClosedAt: nil}

		err := utils.ValidateAccount(acc, "source")

		assert.NoError(t, err)
	})

	t.Run("closed account", func(t *testing.T) {
		now := time.Now()
		acc := &models.Account{ID: 1, IsActive: true, ClosedAt: &now}

		err := utils.ValidateAccount(acc, "source")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("inactive account", func(t *testing.T) {
		acc := &models.Account{ID: 1, IsActive: false, ClosedAt: nil}

		err := utils.ValidateAccount(acc, "destination")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
	})
}

func TestLocalMidnightUTC(t *testing.T) {
	t.Run("converts to midnight in timezone then to UTC", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/New_York")
		input := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

		result := utils.LocalMidnightUTC(input, loc)

		// Midnight in NY (June 15, 2024 00:00 EDT) = 04:00 UTC
		assert.Equal(t, time.UTC, result.Location())
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.June, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("handles UTC location", func(t *testing.T) {
		input := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

		result := utils.LocalMidnightUTC(input, time.UTC)

		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.June, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 0, result.Hour())
	})
}

func TestCalculateNextRun(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name       string
		frequency  string
		current    time.Time
		dayOfMonth int
		want       time.Time
	}{
		{
			name:       "weekly adds 7 days",
			frequency:  "weekly",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.January, 22, 10, 30, 0, 0, loc),
		},
		{
			name:       "biweekly adds 14 days",
			frequency:  "biweekly",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.January, 29, 10, 30, 0, 0, loc),
		},
		{
			name:       "monthly adds 1 month",
			frequency:  "monthly",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.February, 15, 0, 0, 0, 0, loc),
		},
		{
			name:       "quarterly adds 3 months",
			frequency:  "quarterly",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.April, 15, 0, 0, 0, 0, loc),
		},
		{
			name:       "annually adds 1 year",
			frequency:  "annually",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2026, time.January, 15, 0, 0, 0, 0, loc),
		},
		{
			name:       "monthly snaps jan 31 to feb 28",
			frequency:  "monthly",
			current:    time.Date(2025, time.January, 31, 0, 0, 0, 0, loc),
			dayOfMonth: 31,
			want:       time.Date(2025, time.February, 28, 0, 0, 0, 0, loc),
		},
		{
			name:       "monthly recovers back to 31 in march",
			frequency:  "monthly",
			current:    time.Date(2025, time.February, 28, 0, 0, 0, 0, loc),
			dayOfMonth: 31,
			want:       time.Date(2025, time.March, 31, 0, 0, 0, 0, loc),
		},
		{
			name:       "quarterly snaps oct 31 to jan 31",
			frequency:  "quarterly",
			current:    time.Date(2025, time.October, 31, 0, 0, 0, 0, loc),
			dayOfMonth: 31,
			want:       time.Date(2026, time.January, 31, 0, 0, 0, 0, loc),
		},
		{
			name:       "annually snaps feb 29 leap year to feb 28",
			frequency:  "annually",
			current:    time.Date(2024, time.February, 29, 0, 0, 0, 0, loc),
			dayOfMonth: 29,
			want:       time.Date(2025, time.February, 28, 0, 0, 0, 0, loc),
		},
		{
			name:       "unknown defaults to monthly",
			frequency:  "something-else",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.January, 15, 10, 30, 0, 0, loc).AddDate(0, 1, 0),
		},
		{
			name:       "empty defaults to monthly",
			frequency:  "",
			current:    time.Date(2025, time.January, 15, 10, 30, 0, 0, loc),
			dayOfMonth: 15,
			want:       time.Date(2025, time.January, 15, 10, 30, 0, 0, loc).AddDate(0, 1, 0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.CalculateNextRun(tc.current, tc.frequency, tc.dayOfMonth)
			if !got.Equal(tc.want) {
				t.Fatalf("CalculateNextRun(%v, %q) = %v; want %v", tc.current, tc.frequency, got, tc.want)
			}
		})
	}
}
