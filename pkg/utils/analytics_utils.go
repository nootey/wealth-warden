package utils

import (
	"fmt"
	"sort"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
)

func LinearTrend(values []decimal.Decimal) decimal.Decimal {
	n := len(values)
	if n < 2 {
		return decimal.Zero
	}
	nD := decimal.NewFromInt(int64(n))
	var sumX, sumY, sumXY, sumX2 decimal.Decimal
	for i, y := range values {
		x := decimal.NewFromInt(int64(i))
		sumX = sumX.Add(x)
		sumY = sumY.Add(y)
		sumXY = sumXY.Add(x.Mul(y))
		sumX2 = sumX2.Add(x.Mul(x))
	}
	denom := nD.Mul(sumX2).Sub(sumX.Mul(sumX))
	if denom.IsZero() {
		return decimal.Zero
	}
	return nD.Mul(sumXY).Sub(sumX.Mul(sumY)).Div(denom)
}

func TrendDirection(slope decimal.Decimal) string {
	if slope.IsPositive() {
		return "upward"
	}
	if slope.IsNegative() {
		return "downward"
	}
	return "stable"
}

func SignedFixed(d decimal.Decimal) string {
	if d.IsPositive() {
		return "+" + d.StringFixed(2)
	}
	return d.StringFixed(2)
}

func SortedInts(set map[int]struct{}) []int {
	out := make([]int, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// CalendarMonths returns the number of elapsed months to use as the denominator.
// For past years it's always 12; for the current year it's the current month.
func CalendarMonths(year int) int {
	now := time.Now()
	if year < now.Year() {
		return 12
	}
	if year > now.Year() {
		return 12
	}
	return int(now.Month())
}

func TopCategoryChanges(current, comparison []models.YearlyCategoryRow, income bool, limit int) []models.CategoryChange {
	byID := make(map[int64]*models.CategoryChange)
	collect := func(rows []models.YearlyCategoryRow, isCurrent bool) {
		for _, r := range rows {
			text := r.OutflowText
			if income {
				text = r.InflowText
			}
			amount, _ := decimal.NewFromString(text)

			c, ok := byID[r.CategoryID]
			if !ok {
				name := "Uncategorized"
				if r.DisplayName != nil {
					name = *r.DisplayName
				}
				c = &models.CategoryChange{CategoryID: r.CategoryID, Category: name}
				byID[r.CategoryID] = c
			}
			if isCurrent {
				c.Current = amount.Abs()
			} else {
				c.Comparison = amount.Abs()
			}
		}
	}
	collect(current, true)
	collect(comparison, false)

	out := make([]models.CategoryChange, 0, len(byID))
	for _, c := range byID {
		c.Change = c.Current.Sub(c.Comparison)
		if c.Change.IsZero() {
			continue
		}
		if !c.Comparison.IsZero() {
			pct := c.Change.Div(c.Comparison).InexactFloat64() * 100.0
			c.ChangePct = &pct
		}
		out = append(out, *c)
	}

	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i].Change.Abs(), out[j].Change.Abs()
		if !ai.Equal(aj) {
			return ai.GreaterThan(aj)
		}
		return out[i].CategoryID < out[j].CategoryID
	})

	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func YearStrings(years []int) []string {
	out := make([]string, len(years))
	for i, y := range years {
		out[i] = fmt.Sprintf("%d", y)
	}
	return out
}
