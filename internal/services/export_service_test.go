//go:build integration

package services_test

import (
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type ExportServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationExportServiceSuite(t *testing.T) {
	suite.Run(t, new(ExportServiceTestSuite))
}

func (s *ExportServiceTestSuite) TestCreateExport_FailedDispatchStoresClientSafeError() {
	svc := services.NewExportService(
		zaptest.NewLogger(s.T()),
		repositories.NewExportRepository(s.TC.DB),
		repositories.NewTransactionRepository(s.TC.DB),
		repositories.NewAccountRepository(s.TC.DB),
		repositories.NewBalanceRepository(s.TC.DB),
		repositories.NewSettingsRepository(s.TC.DB),
		repositories.NewRulesRepository(s.TC.DB),
		failingDispatcher{},
	)

	_, err := svc.CreateExport(s.Ctx, 1)
	s.Require().Error(err)

	var export models.Export
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Where("user_id = ?", 1).Order("id DESC").First(&export).Error)
	s.Equal(models.ExportStatusFailed, export.Status)
	s.Require().NotNil(export.Error)
	s.Equal(apperr.GenericMessage, *export.Error)
}
