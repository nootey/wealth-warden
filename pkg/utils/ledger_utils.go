package utils

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"wealth-warden/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
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
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func CalculateNextRun(current time.Time, frequency string, dayOfMonth int, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := current.In(loc)

	switch frequency {
	case "monthly":
		next := time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, loc)
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, loc).UTC()
	case "weekly":
		return local.AddDate(0, 0, 7).UTC()
	case "biweekly":
		return local.AddDate(0, 0, 14).UTC()
	case "quarterly":
		next := time.Date(local.Year(), local.Month()+3, 1, 0, 0, 0, 0, loc)
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, loc).UTC()
	case "annually":
		next := time.Date(local.Year()+1, local.Month(), 1, 0, 0, 0, 0, loc)
		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		day := dayOfMonth
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, loc).UTC()
	default:
		return local.AddDate(0, 1, 0).UTC()
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

func CheckGoalAllocation(amountBeingRemoved, uncategorizedBalance decimal.Decimal, classification string) error {
	if strings.EqualFold(classification, "liability") {
		return nil
	}
	if amountBeingRemoved.GreaterThan(uncategorizedBalance) {
		return fmt.Errorf("insufficient free balance: %s available - archive goals or remove contributions to proceed",
			uncategorizedBalance.StringFixed(2))
	}
	return nil
}

func AccountBelowLimit(balance decimal.Decimal, acc *models.Account) bool {
	if acc.AccountType.Classification == "liability" {
		return false
	}
	floor := decimal.Zero
	if acc.CreditLimit != nil {
		floor = acc.CreditLimit.Neg()
	}
	return balance.LessThan(floor)
}

func AccountLimitError(balance decimal.Decimal, acc *models.Account) error {
	if acc.CreditLimit != nil {
		over := balance.Neg().Sub(*acc.CreditLimit)
		return fmt.Errorf("insufficient funds: %s over credit limit", over.StringFixed(2))
	}
	return fmt.Errorf("insufficient funds: resulting balance (%s) would be negative", balance.StringFixed(2))
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsExtremePriceDrop(oldPrice *decimal.Decimal, newPrice decimal.Decimal) bool {
	if oldPrice == nil || oldPrice.IsZero() {
		return false
	}
	if !newPrice.LessThan(*oldPrice) {
		return false
	}
	drop := newPrice.Sub(*oldPrice).Div(*oldPrice).Abs()
	return drop.GreaterThan(decimal.NewFromFloat(0.90))
}

func AddAllocation(buckets map[string]*models.AllocationRow, key, label string, value decimal.Decimal) {
	if bucket, ok := buckets[key]; ok {
		bucket.Value = bucket.Value.Add(value)
		return
	}
	buckets[key] = &models.AllocationRow{Key: key, Label: label, Value: value}
}

func AllocationRows(buckets map[string]*models.AllocationRow, total decimal.Decimal) []models.AllocationRow {
	rows := make([]models.AllocationRow, 0, len(buckets))
	for _, bucket := range buckets {
		if total.IsPositive() {
			bucket.Weight = bucket.Value.Div(total).Round(6)
		}
		rows = append(rows, *bucket)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].Value.Equal(rows[j].Value) {
			return rows[i].Value.GreaterThan(rows[j].Value)
		}
		return rows[i].Key < rows[j].Key
	})

	return rows
}

type TradeExchangeRateKey struct {
	From, To string
	Day      string
}

func NewTradeExchangeRateKey(trade models.InvestmentTrade) TradeExchangeRateKey {
	return TradeExchangeRateKey{
		From: trade.Currency,
		To:   trade.Asset.Account.Currency,
		Day:  trade.TxnDate.UTC().Format("2006-01-02"),
	}
}
