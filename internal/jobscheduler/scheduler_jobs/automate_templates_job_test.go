package scheduler_jobs_test

import (
	"testing"
	"wealth-warden/internal/jobscheduler/scheduler_jobs"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type AutomateTemplateJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAutomateTemplateJobSuite(t *testing.T) {
	suite.Run(t, new(AutomateTemplateJobTestSuite))
}

// Test that automate template job runs
func (s *AutomateTemplateJobTestSuite) TestAutomateTemplateJob_Success() {
	logger := zaptest.NewLogger(s.T())
	job := scheduler_jobs.NewAutomateTemplateJob(logger, s.TC.App, nil, 0)

	err := job.Run(s.Ctx)
	s.NoError(err)
}
