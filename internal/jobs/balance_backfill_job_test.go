package jobs_test

import (
	"testing"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type BalanceBackfillJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBalanceBackfillJobSuite(t *testing.T) {
	suite.Run(t, new(BalanceBackfillJobTestSuite))
}

// Test that backfill job runs
func (s *BalanceBackfillJobTestSuite) TestBalanceBackfillJob_Success() {
	logger := zaptest.NewLogger(s.T())
	job := jobs.NewBalanceBackfillJob(logger, s.TC.App.UserService, s.TC.App.AccountService, 0)

	err := job.Run(s.Ctx)
	s.NoError(err)
}
