package utils_test

import (
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestGetIANATimezones(t *testing.T) {
	timezones := utils.GetIANATimezones()

	t.Run("returns non-empty list", func(t *testing.T) {
		assert.NotEmpty(t, timezones)
		assert.Greater(t, len(timezones), 100, "should contain a comprehensive list of timezones")
	})

	t.Run("contains UTC", func(t *testing.T) {
		assert.Contains(t, timezones, "UTC")
	})

	t.Run("contains major continental timezones", func(t *testing.T) {
		majorTimezones := []string{
			// Africa
			"Africa/Cairo",
			"Africa/Johannesburg",
			"Africa/Lagos",

			// Americas
			"America/New_York",
			"America/Chicago",
			"America/Denver",
			"America/Los_Angeles",
			"America/Toronto",
			"America/Mexico_City",
			"America/Sao_Paulo",

			// Asia
			"Asia/Tokyo",
			"Asia/Shanghai",
			"Asia/Dubai",
			"Asia/Singapore",
			"Asia/Kolkata",

			// Europe
			"Europe/London",
			"Europe/Paris",
			"Europe/Berlin",
			"Europe/Moscow",
			"Europe/Ljubljana",

			// Australia/Pacific
			"Australia/Sydney",
			"Pacific/Auckland",
		}

		for _, tz := range majorTimezones {
			assert.Contains(t, timezones, tz, "should contain major timezone: %s", tz)
		}
	})

	t.Run("all entries follow IANA format", func(t *testing.T) {
		for _, tz := range timezones {
			// Valid IANA timezones are either "UTC" or "Continent/City" format
			if tz != "UTC" {
				assert.Contains(t, tz, "/", "timezone %s should contain '/' separator", tz)
				assert.NotEmpty(t, tz, "timezone should not be empty")
			}
		}
	})

	t.Run("contains no duplicates", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, tz := range timezones {
			assert.False(t, seen[tz], "duplicate timezone found: %s", tz)
			seen[tz] = true
		}
	})

	t.Run("contains regional coverage", func(t *testing.T) {
		regions := map[string]bool{
			"Africa":    false,
			"America":   false,
			"Asia":      false,
			"Australia": false,
			"Europe":    false,
			"Pacific":   false,
			"Atlantic":  false,
		}

		for _, tz := range timezones {
			for region := range regions {
				if len(tz) > len(region) && tz[:len(region)] == region {
					regions[region] = true
				}
			}
		}

		for region, found := range regions {
			assert.True(t, found, "should contain at least one timezone from %s", region)
		}
	})

	t.Run("returns same list on multiple calls", func(t *testing.T) {
		timezones1 := utils.GetIANATimezones()
		timezones2 := utils.GetIANATimezones()

		assert.Equal(t, timezones1, timezones2, "should return consistent results")
	})
}
