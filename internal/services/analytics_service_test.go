package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type AnalyticsServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAnalyticsServiceSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsServiceTestSuite))
}

// closedAccount creates and immediately closes an empty account, returning its id.
func (s *AnalyticsServiceTestSuite) closedAccount(userID int64) int64 {
	accSvc := s.TC.App.AccountService
	zero := decimal.Zero

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Closed Analytics Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &zero,
		OpenedAt:       time.Now(),
	})
	s.Require().NoError(err)
	s.Require().NoError(accSvc.CloseAccount(s.Ctx, userID, accID))

	return accID
}

func (s *AnalyticsServiceTestSuite) checkingWithIncome(userID int64, amount int64, dates ...time.Time) int64 {
	zero := decimal.Zero
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Analytics Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &zero,
		OpenedAt:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	s.Require().NoError(err)

	for _, d := range dates {
		_, err := s.TC.App.TransactionService.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
			AccountID: accID,
			Direction: "income",
			Amount:    decimal.NewFromInt(amount),
			TxnDate:   d,
		})
		s.Require().NoError(err)
	}

	return accID
}

func (s *AnalyticsServiceTestSuite) TestGenerateCategoryReport_ValidationError_RequiresPrimaryWhenSecondarySet() {
	svc := s.TC.App.AnalyticsService
	_, err := svc.GenerateCategoryReport(s.Ctx, 1, models.CategoryReportParams{
		OutflowCategoryIDs: []int64{1},
		InflowCategoryIDs:  nil,
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "primary category")
}

func (s *AnalyticsServiceTestSuite) TestGenerateCategoryReport_CreatesReportWithPendingStatus() {
	svc := s.TC.App.AnalyticsService
	params := models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
	}
	report, err := svc.GenerateCategoryReport(s.Ctx, 1, params)
	s.Require().NoError(err)
	s.Require().NotNil(report)
	s.Equal("pending", report.Status)
	s.Equal("category", report.Type)
	s.Positive(report.ID)
}

func (s *AnalyticsServiceTestSuite) TestGenerateCategoryReport_DescriptionDoesNotOverrideName() {
	svc := s.TC.App.AnalyticsService
	params := models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
		Description:       "monster",
	}
	report, err := svc.GenerateCategoryReport(s.Ctx, 1, params)
	s.Require().NoError(err)
	s.Contains(report.Name, "2024")
	s.NotEqual("monster", report.Name)
}

func (s *AnalyticsServiceTestSuite) TestGenerateCategoryReport_NameFromYears() {
	svc := s.TC.App.AnalyticsService
	params := models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2023, 2024},
	}
	report, err := svc.GenerateCategoryReport(s.Ctx, 1, params)
	s.Require().NoError(err)
	s.Contains(report.Name, "2023")
	s.Contains(report.Name, "2024")
}

func (s *AnalyticsServiceTestSuite) TestListReportsPaginated_EmptyForUnknownUser() {
	svc := s.TC.App.AnalyticsService
	reports, paginator, err := svc.ListReportsPaginated(s.Ctx, 99999, utils.PaginationParams{
		PageNumber:  1,
		RowsPerPage: 10,
	})
	s.Require().NoError(err)
	s.Empty(reports)
	s.Equal(0, paginator.TotalRecords)
}

func (s *AnalyticsServiceTestSuite) TestListReportsPaginated_PaginatorFields() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)

	for range 3 {
		_, err := svc.GenerateCategoryReport(s.Ctx, userID, models.CategoryReportParams{
			InflowCategoryIDs: []int64{1},
			Years:             []int{2024},
		})
		s.Require().NoError(err)
	}

	reports, paginator, err := svc.ListReportsPaginated(s.Ctx, userID, utils.PaginationParams{
		PageNumber:  1,
		RowsPerPage: 2,
	})
	s.Require().NoError(err)
	s.Len(reports, 2)
	s.Equal(1, paginator.CurrentPage)
	s.Equal(2, paginator.RowsPerPage)
	s.Equal(1, paginator.From)
	s.Equal(2, paginator.To)
	s.GreaterOrEqual(paginator.TotalRecords, 3)
}

func (s *AnalyticsServiceTestSuite) TestDeleteReport_RemovesRecord() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)

	report, err := svc.GenerateCategoryReport(s.Ctx, userID, models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
	})
	s.Require().NoError(err)

	err = svc.DeleteReport(s.Ctx, userID, report.ID)
	s.Require().NoError(err)

	_, err = svc.FindReportByID(s.Ctx, report.ID, userID)
	s.Require().Error(err)
}

func (s *AnalyticsServiceTestSuite) TestDeleteReport_NotFound_ReturnsError() {
	svc := s.TC.App.AnalyticsService
	err := svc.DeleteReport(s.Ctx, 1, 99999)
	s.Require().Error(err)
}

func (s *AnalyticsServiceTestSuite) TestGetYearlyCashFlowBreakdown_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	_, err := svc.GetYearlyCashFlowBreakdown(s.Ctx, userID, time.Now().Year(), &accID)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

func (s *AnalyticsServiceTestSuite) TestGetYearlySankeyData_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	_, err := svc.GetYearlySankeyData(s.Ctx, userID, &accID, time.Now().Year())

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

func (s *AnalyticsServiceTestSuite) TestGetAccountBasicStatistics_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	_, err := svc.GetAccountBasicStatistics(s.Ctx, &accID, userID, time.Now().Year())

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

func (s *AnalyticsServiceTestSuite) TestGetMonthlyStats_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	now := time.Now()
	_, err := svc.GetMonthlyStats(s.Ctx, userID, &accID, now.Year(), int(now.Month()))

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

func (s *AnalyticsServiceTestSuite) TestGetTodayStats_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	_, err := svc.GetTodayStats(s.Ctx, userID, &accID)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

// Exercises getYearStatsWithAllocations, which is unexported and only reachable
// through this wrapper.
func (s *AnalyticsServiceTestSuite) TestGetYearlyBreakdownStats_RejectsClosedAccount() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.closedAccount(userID)

	_, err := svc.GetYearlyBreakdownStats(s.Ctx, &accID, userID, time.Now().Year(), nil)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "closed")
}

func (s *AnalyticsServiceTestSuite) TestGetAccountBasicStatistics_AveragesTakeHomeOverActiveMonths() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	accID := s.checkingWithIncome(userID, 1000,
		time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 4, 10, 0, 0, 0, 0, time.UTC),
	)

	stats, err := svc.GetAccountBasicStatistics(s.Ctx, &accID, userID, 2024)

	s.Require().NoError(err)
	s.Assert().True(stats.AvgMonthlyTakeHome.Equal(decimal.NewFromInt(1000)), "got %s", stats.AvgMonthlyTakeHome)
	s.Assert().True(stats.AvgMonthlyTakeHome.Equal(stats.AvgMonthlyInflow))
}

func (s *AnalyticsServiceTestSuite) TestGetYearlyBreakdownStats_AveragesTakeHomeOverElapsedMonths() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	accID := s.checkingWithIncome(userID, 1200, today)

	res, err := svc.GetYearlyBreakdownStats(s.Ctx, &accID, userID, today.Year(), nil)

	s.Require().NoError(err)
	cur := res.CurrentYear
	s.Assert().True(cur.AvgMonthlyTakeHome.Equal(cur.AvgMonthlyInflow), "take home %s, inflow %s", cur.AvgMonthlyTakeHome, cur.AvgMonthlyInflow)
}

func (s *AnalyticsServiceTestSuite) TestGetYearlyBreakdownStats_CategoryChangesCapBothYearsAtCurrentMonth() {
	svc := s.TC.App.AnalyticsService
	userID := int64(1)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	lastYear := today.Year() - 1
	accID := s.checkingWithIncome(userID, 1000, time.Date(lastYear, 1, 10, 0, 0, 0, 0, time.UTC))

	for amount, date := range map[int64]time.Time{
		5000: time.Date(lastYear, 12, 10, 0, 0, 0, 0, time.UTC),
		1200: today,
	} {
		_, err := s.TC.App.TransactionService.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
			AccountID: accID,
			Direction: "income",
			Amount:    decimal.NewFromInt(amount),
			TxnDate:   date,
		})
		s.Require().NoError(err)
	}

	res, err := svc.GetYearlyBreakdownStats(s.Ctx, &accID, userID, today.Year(), nil)

	s.Require().NoError(err)
	s.Require().NotNil(res.CategoryChanges)
	s.Assert().Equal(int(today.Month()), res.CategoryChanges.ThroughMonth)
	s.Assert().Empty(res.CategoryChanges.Expense)
	s.Require().Len(res.CategoryChanges.Income, 1)

	wantComparison := decimal.NewFromInt(1000)
	if today.Month() == time.December {
		wantComparison = decimal.NewFromInt(6000)
	}
	inc := res.CategoryChanges.Income[0]
	s.Assert().True(inc.Current.Equal(decimal.NewFromInt(1200)), "current %s", inc.Current)
	s.Assert().True(inc.Comparison.Equal(wantComparison), "comparison %s", inc.Comparison)
}
