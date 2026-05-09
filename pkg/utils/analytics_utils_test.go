package utils_test

import (
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

func d(v int) decimal.Decimal { return decimal.NewFromInt(int64(v)) }

func TestLinearTrend_EmptyAndSingle(t *testing.T) {
	if got := utils.LinearTrend(nil); !got.IsZero() {
		t.Errorf("LinearTrend(nil) = %s, want 0", got)
	}
	if got := utils.LinearTrend([]decimal.Decimal{d(5)}); !got.IsZero() {
		t.Errorf("LinearTrend(single) = %s, want 0", got)
	}
}

func TestLinearTrend_Ascending(t *testing.T) {
	values := []decimal.Decimal{d(1), d(2), d(3), d(4)}
	if got := utils.LinearTrend(values); !got.IsPositive() {
		t.Errorf("LinearTrend(ascending) = %s, want positive", got)
	}
}

func TestLinearTrend_Descending(t *testing.T) {
	values := []decimal.Decimal{d(4), d(3), d(2), d(1)}
	if got := utils.LinearTrend(values); !got.IsNegative() {
		t.Errorf("LinearTrend(descending) = %s, want negative", got)
	}
}

func TestLinearTrend_Flat(t *testing.T) {
	values := []decimal.Decimal{d(5), d(5), d(5), d(5)}
	if got := utils.LinearTrend(values); !got.IsZero() {
		t.Errorf("LinearTrend(flat) = %s, want 0", got)
	}
}

func TestTrendDirection(t *testing.T) {
	cases := []struct {
		slope decimal.Decimal
		want  string
	}{
		{decimal.NewFromInt(3), "upward"},
		{decimal.NewFromInt(-2), "downward"},
		{decimal.Zero, "stable"},
	}
	for _, c := range cases {
		if got := utils.TrendDirection(c.slope); got != c.want {
			t.Errorf("TrendDirection(%s) = %q, want %q", c.slope, got, c.want)
		}
	}
}

func TestSignedFixed(t *testing.T) {
	cases := []struct {
		input decimal.Decimal
		want  string
	}{
		{decimal.NewFromFloat(3.5), "+3.50"},
		{decimal.NewFromFloat(-2.1), "-2.10"},
		{decimal.Zero, "0.00"},
	}
	for _, c := range cases {
		if got := utils.SignedFixed(c.input); got != c.want {
			t.Errorf("SignedFixed(%s) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSortedInts(t *testing.T) {
	in := map[int]struct{}{3: {}, 1: {}, 5: {}, 2: {}}
	got := utils.SortedInts(in)
	want := []int{1, 2, 3, 5}
	if len(got) != len(want) {
		t.Fatalf("SortedInts len = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("SortedInts[%d] = %d, want %d", i, got[i], v)
		}
	}
}

func TestSortedInts_Empty(t *testing.T) {
	if got := utils.SortedInts(map[int]struct{}{}); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestYearStrings(t *testing.T) {
	got := utils.YearStrings([]int{2023, 2024, 2025})
	want := []string{"2023", "2024", "2025"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("YearStrings[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestCalendarMonths_PastYear(t *testing.T) {
	if got := utils.CalendarMonths(2000); got != 12 {
		t.Errorf("CalendarMonths(2000) = %d, want 12", got)
	}
}

func TestCalendarMonths_FutureYear(t *testing.T) {
	if got := utils.CalendarMonths(9999); got != 12 {
		t.Errorf("CalendarMonths(9999) = %d, want 12", got)
	}
}
