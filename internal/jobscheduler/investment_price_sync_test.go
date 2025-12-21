package jobscheduler_test

import (
	"testing"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
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
	job := jobscheduler.NewInvestmentPriceSyncJob(logger, s.TC.App)

	err := job.Run(s.Ctx)
	s.NoError(err)
}
