package jobscheduler_test

import (
	"testing"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type BackfillJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBackfillJobSuite(t *testing.T) {
	suite.Run(t, new(BackfillJobTestSuite))
}

// Test that backfill job runs
func (s *BackfillJobTestSuite) TestBackfillJob_Success() {
	logger := zaptest.NewLogger(s.T())
	job := jobscheduler.NewBackfillJob(logger, s.TC.App)

	err := job.Run(s.Ctx)
	s.NoError(err)
}
