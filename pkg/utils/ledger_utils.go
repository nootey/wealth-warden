package utils

import (
	"errors"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
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
	local := t.In(loc)
	y, m, d := local.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
}

func CalculateNextRun(current time.Time, frequency string, dayOfMonth int) time.Time {
	switch frequency {
	case "monthly":
		next := time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location())
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, current.Location()).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, current.Location())
	case "weekly":
		return current.AddDate(0, 0, 7)
	case "biweekly":
		return current.AddDate(0, 0, 14)
	case "quarterly":
		next := time.Date(current.Year(), current.Month()+3, 1, 0, 0, 0, 0, current.Location())
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, current.Location()).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, current.Location())
	case "annually":
		next := time.Date(current.Year()+1, current.Month(), 1, 0, 0, 0, 0, current.Location())
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, current.Location()).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, current.Location())
	default:
		return current.AddDate(0, 1, 0) // default to monthly
	}
}

func CategorizeTransferDestination(accountType *models.AccountType) (isSavings, isInvestment, isDebt bool) {
	if accountType == nil {
		return false, false, false
	}

	subtype := accountType.Subtype
	accType := accountType.Type
	classification := accountType.Classification

	// Savings category
	if subtype == "savings" || subtype == "health_savings" || subtype == "money_market" {
		return true, false, false
	}

	// Investment category
	if accType == "investment" || accType == "crypto" {
		return false, true, false
	}

	// Debt category
	if classification == "liability" {
		return false, false, true
	}

	return false, false, false
}

func AdjustToWeekday(date time.Time) time.Time {
	weekday := date.Weekday()

	switch weekday {
	case time.Saturday:
		return date.AddDate(0, 0, 2)
	case time.Sunday:
		return date.AddDate(0, 0, 1)
	default:
		return date
	}
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
