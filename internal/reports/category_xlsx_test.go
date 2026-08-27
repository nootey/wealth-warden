package reports_test

import (
	"bytes"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/reports"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

func row(year, month int, name, class string, total int64) models.CategoryReportDataRow {
	return models.CategoryReportDataRow{
		Year: year, Month: month,
		CategoryName: name, Classification: class,
		Total: decimal.NewFromInt(total), TxnCount: 1,
	}
}

func build(t *testing.T, rows []models.CategoryReportDataRow, params models.CategoryReportParams) *excelize.File {
	t.Helper()
	data, err := reports.BuildCategoryXLSX(rows, params, "All accounts")
	if err != nil {
		t.Fatalf("BuildCategoryXLSX: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("the workbook does not open: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func cell(t *testing.T, f *excelize.File, sheet, ref string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, ref)
	if err != nil {
		t.Fatalf("GetCellValue(%s!%s): %v", sheet, ref, err)
	}
	return v
}

func TestBuildCategoryXLSX_SheetsPerYear(t *testing.T) {
	rows := []models.CategoryReportDataRow{
		row(2022, 1, "Salary", "inflow", 4000),
		row(2023, 1, "Salary", "inflow", 4500),
		row(2024, 1, "Salary", "inflow", 5000),
	}

	t.Run("all time adds a comparison sheet", func(t *testing.T) {
		f := build(t, rows, models.CategoryReportParams{AllTime: true})
		want := []string{"Summary", "All Time", "2022", "2023", "2024"}
		if got := f.GetSheetList(); !equal(got, want) {
			t.Errorf("sheets = %v, want %v", got, want)
		}
	})

	t.Run("single year has no comparison sheet", func(t *testing.T) {
		f := build(t, rows[2:], models.CategoryReportParams{Years: []int{2024}})
		want := []string{"Summary", "2024"}
		if got := f.GetSheetList(); !equal(got, want) {
			t.Errorf("sheets = %v, want %v", got, want)
		}
	})
}

// Effective is primary minus secondary. Getting the sign or the grouping wrong
// is the failure that silently produces a plausible but wrong report.
func TestBuildCategoryXLSX_SummaryEffective(t *testing.T) {
	rows := []models.CategoryReportDataRow{
		row(2024, 1, "Salary", "inflow", 5000),
		row(2024, 2, "Salary", "inflow", 5200),
		row(2024, 1, "Rent", "outflow", 1200),
	}
	f := build(t, rows, models.CategoryReportParams{Years: []int{2024}})

	if got := cell(t, f, "Summary", "B3"); got != "All accounts" {
		t.Errorf("scope label = %q, want \"All accounts\"", got)
	}
	for _, tc := range []struct{ ref, want string }{
		{"A6", "2024"},
		{"B6", "10200"}, // primary
		{"C6", "1200"},  // secondary
		{"D6", "9000"},  // effective
		{"G6", "2"},     // active months
	} {
		if got := cell(t, f, "Summary", tc.ref); got != tc.want {
			t.Errorf("Summary!%s = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
