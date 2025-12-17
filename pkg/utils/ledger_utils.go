package utils

import (
	"fmt"
	"time"
	"wealth-warden/internal/models"
)

func ValidateAccount(acc *models.Account, role string) error {
	if acc.ClosedAt != nil {
		return fmt.Errorf("%s account (ID=%d) is closed and cannot be used", role, acc.ID)
	}
	if !acc.IsActive {
		return fmt.Errorf("%s account (ID=%d) is inactive and cannot be used", role, acc.ID)
	}
	return nil
}

func LocalMidnightUTC(t time.Time, loc *time.Location) time.Time {
	lt := t.In(loc)
	lm := time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, loc)
	return lm.UTC()
}

func CalculateNextRun(current time.Time, frequency string) time.Time {
	switch frequency {
	case "weekly":
		return current.AddDate(0, 0, 7)
	case "biweekly":
		return current.AddDate(0, 0, 14)
	case "monthly":
		return current.AddDate(0, 1, 0)
	case "quarterly":
		return current.AddDate(0, 3, 0)
	case "annually":
		return current.AddDate(1, 0, 0)
	default:
		return current.AddDate(0, 1, 0) // default to monthly
	}
}
