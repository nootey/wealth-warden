package services_test

import (
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/suite"
)

type AnalyticsServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAnalyticsServiceSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsServiceTestSuite))
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

func (s *AnalyticsServiceTestSuite) TestGenerateCategoryReport_NameFromDescription() {
	svc := s.TC.App.AnalyticsService
	params := models.CategoryReportParams{
		InflowCategoryIDs: []int64{1},
		Years:             []int{2024},
		Description:       "My Custom Report",
	}
	report, err := svc.GenerateCategoryReport(s.Ctx, 1, params)
	s.Require().NoError(err)
	s.Equal("My Custom Report", report.Name)
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
