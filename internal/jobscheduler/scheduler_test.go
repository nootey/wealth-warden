package jobscheduler_test

import (
	"context"
	"testing"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type SchedulerTestSuite struct {
	suite.Suite
	logger    *zap.Logger
	container *bootstrap.ServiceContainer
	scheduler *jobscheduler.Scheduler
}

func (suite *SchedulerTestSuite) SetupTest() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.container = &bootstrap.ServiceContainer{
		Config: &config.Config{
			FinanceAPIBaseURL: "https://query1.finance.yahoo.com",
		},
	}

	var err error
	suite.scheduler, err = jobscheduler.NewScheduler(suite.logger, suite.container, jobscheduler.SchedulerConfig{
		StartBalanceBackfillImmediately:      false,
		StartTemplatesImmediately:            false,
		StartAssetPriceSyncImmediately:       false,
		StartAssetHistoryBackfillImmediately: false,
	})
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := suite.scheduler.Start(ctx)
	suite.NoError(err)

	err = suite.scheduler.Shutdown()
	suite.NoError(err)
}

// Test creating scheduler with nil logger returns error
func (suite *SchedulerTestSuite) TestScheduler_NewWithNilLogger() {
	scheduler, err := jobscheduler.NewScheduler(nil, suite.container, jobscheduler.SchedulerConfig{
		StartBalanceBackfillImmediately:      false,
		StartTemplatesImmediately:            false,
		StartAssetPriceSyncImmediately:       false,
		StartAssetHistoryBackfillImmediately: false,
	})
	suite.Error(err)
	suite.Nil(scheduler)
}

// Test creating scheduler with nil container returns error
func (suite *SchedulerTestSuite) TestScheduler_NewWithNilContainer() {
	scheduler, err := jobscheduler.NewScheduler(suite.logger, nil, jobscheduler.SchedulerConfig{
		StartBalanceBackfillImmediately:      false,
		StartTemplatesImmediately:            false,
		StartAssetPriceSyncImmediately:       false,
		StartAssetHistoryBackfillImmediately: false,
	})
	suite.Error(err)
	suite.Nil(scheduler)
	suite.Contains(err.Error(), "container cannot be nil")
}

func TestSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTestSuite))
}
