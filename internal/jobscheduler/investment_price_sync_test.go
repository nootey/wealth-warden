package jobscheduler_test

import (
	"testing"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/prices"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type InvestmentPriceSyncJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestInvestmentPriceSyncJobSuite(t *testing.T) {
	suite.Run(t, new(InvestmentPriceSyncJobTestSuite))
}

// Test that job runs
func (s *InvestmentPriceSyncJobTestSuite) TestInvestmentPriceSyncJob_Success() {
	logger := zaptest.NewLogger(s.T())

	client, err := prices.NewPriceFetchClient(s.TC.App.Config.FinanceAPIBaseURL)
	if err != nil {
		logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	job := jobscheduler.NewInvestmentPriceSyncJob(logger, s.TC.App, client)

	err = job.Run(s.Ctx)
	s.NoError(err)
}
