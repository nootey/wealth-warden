package utils

import (
	"fmt"
	"sort"
	"time"

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

func YearStrings(years []int) []string {
	out := make([]string, len(years))
	for i, y := range years {
		out[i] = fmt.Sprintf("%d", y)
	}
	return out
}
