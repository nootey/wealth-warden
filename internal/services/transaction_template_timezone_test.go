package services

import (
	"testing"
	"time"
	"wealth-warden/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReanchorTemplates_SameLocalDate(t *testing.T) {
	// next_run_at is midnight UTC (00:00 UTC = 01:00 Ljubljana CET).
	// Switching to Ljubljana should produce 2025-02-15 23:00 UTC (midnight Ljubljana).
	ljubljana, err := time.LoadLocation("Europe/Ljubljana")
	require.NoError(t, err)

	updates := reanchorTemplates([]models.TransactionTemplate{
		{ID: 1, NextRunAt: time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC), DayOfMonth: 15},
	}, ljubljana)

	require.Len(t, updates, 1)
	u := updates[0]
	assert.Equal(t, int64(1), u.ID)
	assert.Equal(t, 15, u.DayOfMonth, "the anchor day must survive the move")

	local := u.NextRunAt.In(ljubljana)
	assert.Equal(t, 0, local.Hour())
	assert.Equal(t, 0, local.Minute())
	assert.Equal(t, 15, local.Day())
	assert.Equal(t, time.February, local.Month())
}

func TestReanchorTemplates_MultipleTemplates(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)

	// Midday UTC: the local Paris date matches the UTC date whatever the DST offset.
	updates := reanchorTemplates([]models.TransactionTemplate{
		{ID: 1, NextRunAt: time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC), DayOfMonth: 10},
		{ID: 2, NextRunAt: time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC), DayOfMonth: 20},
	}, paris)

	require.Len(t, updates, 2)
	for _, u := range updates {
		local := u.NextRunAt.In(paris)
		assert.Equal(t, 0, local.Hour(), "template %d: expected midnight Paris", u.ID)
		assert.Equal(t, 0, local.Minute())
	}
	assert.Equal(t, 10, updates[0].DayOfMonth)
	assert.Equal(t, 20, updates[1].DayOfMonth)
}

// An anchor day past the end of a short month keeps its intent: DayOfMonth stays
// 31 even though next_run_at lands on the 28th.
func TestReanchorTemplates_ShortMonthKeepsAnchorDay(t *testing.T) {
	updates := reanchorTemplates([]models.TransactionTemplate{
		{ID: 1, NextRunAt: time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC), DayOfMonth: 31},
	}, time.UTC)

	require.Len(t, updates, 1)
	assert.Equal(t, 31, updates[0].DayOfMonth)
	assert.Equal(t, 28, updates[0].NextRunAt.Day())
}
