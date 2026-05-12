package queue_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"

	"github.com/shopspring/decimal"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
)

type mockAnalyticsRepo struct {
	fetchRows   []models.CategoryReportDataRow
	fetchErr    error
	updateErr   error
	updateErrOn int // which call (1-indexed) should fail; 0 = always fail
	updateCalls int
	updates     []map[string]interface{}
}

func (m *mockAnalyticsRepo) UpdateReport(_ context.Context, _ *gorm.DB, _ int64, fields map[string]interface{}) error {
	m.updateCalls++
	m.updates = append(m.updates, fields)
	if m.updateErr != nil && (m.updateErrOn == 0 || m.updateCalls == m.updateErrOn) {
		return m.updateErr
	}
	return nil
}

func (m *mockAnalyticsRepo) FetchCategoryReportData(_ context.Context, _ *gorm.DB, _ int64, _, _ []int64, _ []int, _ bool, _ string) ([]models.CategoryReportDataRow, error) {
	return m.fetchRows, m.fetchErr
}

func (m *mockAnalyticsRepo) BeginTx(_ context.Context) (*gorm.DB, error) { return nil, nil }
func (m *mockAnalyticsRepo) CountReports(_ context.Context, _ *gorm.DB, _ int64) (int64, error) {
	return 0, nil
}
func (m *mockAnalyticsRepo) FindReports(_ context.Context, _ *gorm.DB, _ int64, _, _ int) ([]models.Report, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FindReportByID(_ context.Context, _ *gorm.DB, _, _ int64) (*models.Report, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) InsertReport(_ context.Context, _ *gorm.DB, _ *models.Report) error {
	return nil
}
func (m *mockAnalyticsRepo) DeleteReport(_ context.Context, _ *gorm.DB, _, _ int64) error {
	return nil
}
func (m *mockAnalyticsRepo) FetchNetWorthSeries(_ context.Context, _ *gorm.DB, _ int64, _ string, _, _ time.Time, _ string, _ *int64) ([]models.ChartPoint, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchLatestNetWorth(_ context.Context, _ *gorm.DB, _ int64, _ string, _ *int64) (time.Time, string, error) {
	return time.Time{}, "", nil
}
func (m *mockAnalyticsRepo) FetchDailyTotals(_ context.Context, _ *gorm.DB, _ int64, _ *int64, _ time.Time) (*models.MonthlyTotalsRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchDailyTotalsCheckingOnly(_ context.Context, _ *gorm.DB, _ int64, _ []int64, _ time.Time) (*models.MonthlyTotalsRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchYearlyTotals(_ context.Context, _ *gorm.DB, _ int64, _ *int64, _ int) (models.YearlyTotalsRow, error) {
	return models.YearlyTotalsRow{}, nil
}
func (m *mockAnalyticsRepo) FetchYearlyCategoryTotals(_ context.Context, _ *gorm.DB, _ int64, _ *int64, _ int) ([]models.YearlyCategoryRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchMonthlyCategoryTotals(_ context.Context, _ *gorm.DB, _ int64, _ *int64, _, _ int) ([]models.YearlyCategoryRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchMonthlyTotals(_ context.Context, _ *gorm.DB, _ int64, _ *int64, _ int) ([]models.MonthlyTotalsRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchMonthlyTotalsCheckingOnly(_ context.Context, _ *gorm.DB, _ int64, _ []int64, _ int) ([]models.MonthlyTotalsRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) FetchMonthlyCategoryTotalsCheckingOnly(_ context.Context, _ *gorm.DB, _ int64, _ []int64, _, _ int) ([]models.YearlyCategoryRow, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetAvailableStatsYears(_ context.Context, _ *gorm.DB, _ *int64, _ int64, _ bool) ([]models.AvailableStatsYear, error) {
	return nil, nil
}

var sampleRows = []models.CategoryReportDataRow{
	{Year: 2024, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5000)},
	{Year: 2024, Month: 2, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5200)},
	{Year: 2024, Month: 3, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5100)},
}

func cleanupReportFiles(t *testing.T, userID string) {
	t.Helper()
	t.Cleanup(func() {
		err := os.RemoveAll(filepath.Join("storage", "reports", userID))
		if err != nil {
			fmt.Println(err.Error())
		}
	})
}

func TestGenerateCategoryReportJob_HappyPath(t *testing.T) {
	repo := &mockAnalyticsRepo{fetchRows: sampleRows}
	job := queue.NewGenerateCategoryReportJob(zaptest.NewLogger(t), repo, 1, 1, models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
	})
	cleanupReportFiles(t, "1")

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateCalls < 2 {
		t.Errorf("expected at least 2 UpdateReport calls, got %d", repo.updateCalls)
	}
	if s, _ := repo.updates[0]["status"].(string); s != "processing" {
		t.Errorf("first update status = %q, want \"processing\"", s)
	}
	last := repo.updates[len(repo.updates)-1]
	if s, _ := last["status"].(string); s != "completed" {
		t.Errorf("last update status = %q, want \"completed\"", s)
	}
	if last["file_path"] == nil {
		t.Error("expected file_path to be set on completion")
	}
}

func TestGenerateCategoryReportJob_FetchError_SetsFailedStatus(t *testing.T) {
	repo := &mockAnalyticsRepo{fetchErr: errors.New("db unavailable")}
	job := queue.NewGenerateCategoryReportJob(zaptest.NewLogger(t), repo, 42, 1, models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
	})

	if err := job.Process(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, u := range repo.updates {
		if s, _ := u["status"].(string); s == "failed" {
			return
		}
	}
	t.Error("expected a failed status update")
}

func TestGenerateCategoryReportJob_InitialUpdateError_ReturnsImmediately(t *testing.T) {
	repo := &mockAnalyticsRepo{updateErr: errors.New("write failed"), updateErrOn: 1}
	job := queue.NewGenerateCategoryReportJob(zaptest.NewLogger(t), repo, 1, 1, models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
	})

	if err := job.Process(context.Background()); err == nil {
		t.Error("expected error when initial UpdateReport fails")
	}
	if repo.updateCalls != 1 {
		t.Errorf("expected exactly 1 UpdateReport call, got %d", repo.updateCalls)
	}
}

func TestGenerateCategoryReportJob_AllTime_MultipleYears(t *testing.T) {
	rows := []models.CategoryReportDataRow{
		{Year: 2022, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(4000)},
		{Year: 2023, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(4500)},
		{Year: 2024, Month: 1, CategoryName: "Salary", Classification: "inflow", Total: decimal.NewFromInt(5000)},
	}
	repo := &mockAnalyticsRepo{fetchRows: rows}
	job := queue.NewGenerateCategoryReportJob(zaptest.NewLogger(t), repo, 2, 1, models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		AllTime:           true,
	})
	cleanupReportFiles(t, "1")

	if err := job.Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := repo.updates[len(repo.updates)-1]
	if s, _ := last["status"].(string); s != "completed" {
		t.Errorf("status = %q, want \"completed\"", s)
	}
}
