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
