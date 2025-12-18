package jobscheduler_test

import (
	"testing"
	"wealth-warden/internal/jobscheduler"
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
	job := jobscheduler.NewAutomateTemplateJob(logger, s.TC.App)

	err := job.Run(s.Ctx)
	s.NoError(err)
}
