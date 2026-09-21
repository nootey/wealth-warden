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

func NormalizeDescription(description string) string {
	return strings.ToLower(strings.Join(strings.Fields(description), " "))
}

func ContentFingerprint(day time.Time, direction string, amount decimal.Decimal, currency, description string) string {
	return strings.Join([]string{
		day.UTC().Format("2006-01-02"),
		direction,
		amount.StringFixed(4),
		currency,
		NormalizeDescription(description),
	}, "|")
}

func ConsumeDuplicate(counts map[string]int, fp string) bool {
	if counts[fp] > 0 {
		counts[fp]--
		return true
	}
	return false
}

func PartialMatchKey(direction, currency string, amount decimal.Decimal) string {
	return strings.Join([]string{direction, amount.StringFixed(4), currency}, "|")
}

func DayDiff(a, b time.Time) int {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	da := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	db := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	d := int(da.Sub(db).Hours() / 24)
	if d < 0 {
		d = -d
	}
	return d
}
