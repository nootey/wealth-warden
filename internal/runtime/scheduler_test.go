package runtime_test

import (
	"testing"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/runtime"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type SchedulerTestSuite struct {
	suite.Suite
	logger    *zap.Logger
	container *bootstrap.Container
	scheduler *runtime.Scheduler
}

func (suite *SchedulerTestSuite) SetupTest() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.container = &bootstrap.Container{
		Config: &config.Config{
			FinanceAPIBaseURL: "https://query1.finance.yahoo.com",
		},
	}

	var err error
	suite.scheduler, err = runtime.NewScheduler(suite.logger, suite.container, false)
	suite.NoError(err)
	suite.NotNil(suite.scheduler)
}

func (suite *SchedulerTestSuite) TearDownTest() {
	if suite.scheduler != nil {
		_ = suite.scheduler.Shutdown()
	}
}

// Test that scheduler is created successfully
func (suite *SchedulerTestSuite) TestScheduler_New() {
	suite.NotNil(suite.scheduler)
}

// Test that scheduler can start and shutdown
func (suite *SchedulerTestSuite) TestScheduler_StartAndShutdown() {
	err := suite.scheduler.Start()
	suite.NoError(err)

	err = suite.scheduler.Shutdown()
	suite.NoError(err)
}

// Test creating scheduler with nil logger returns error
func (suite *SchedulerTestSuite) TestScheduler_NewWithNilLogger() {
	scheduler, err := runtime.NewScheduler(nil, suite.container, false)
	suite.Error(err)
	suite.Nil(scheduler)
}

// Test creating scheduler with nil container returns error
func (suite *SchedulerTestSuite) TestScheduler_NewWithNilContainer() {
	scheduler, err := runtime.NewScheduler(suite.logger, nil, false)
	suite.Error(err)
	suite.Nil(scheduler)
	suite.Contains(err.Error(), "container cannot be nil")
}

func TestSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTestSuite))
}
