package utils

import (
	"github.com/shopspring/decimal"
	"strings"
	"time"
	"wealth-warden/internal/models"
)

func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return CleanString(*s).(string)
}

// CleanString trims leading/trailing spaces for both `string` and `*string` types.
// - If a string is passed, it returns a cleaned string.
// - If a *string is passed, it returns a cleaned *string (or nil if the input was nil).
func CleanString(input interface{}) interface{} {
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v)
	case *string:
		if v == nil {
			return nil
		}
		cleaned := strings.TrimSpace(*v)
		return &cleaned
	default:
		return input
	}
}

func NormalizeName(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), " ", "_")
	s = strings.ReplaceAll(strings.ToLower(s), ":", "_")
	return s
}

func ParseStates(raw []string) []string {
	var out []string
	for _, v := range raw {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func ValidJobState(state string) bool {
	for _, s := range models.RiverJobStates {
		if s == state {
			return true
		}
	}
	return false
}

func ContentFingerprint(day time.Time, direction string, amount decimal.Decimal, currency, description string) string {
	normDesc := strings.ToLower(strings.Join(strings.Fields(description), " "))
	return strings.Join([]string{
		day.UTC().Format("2006-01-02"),
		direction,
		amount.StringFixed(4),
		currency,
		normDesc,
	}, "|")
}

func ConsumeDuplicate(counts map[string]int, fp string) bool {
	if counts[fp] > 0 {
		counts[fp]--
		return true
	}
	return false
}
