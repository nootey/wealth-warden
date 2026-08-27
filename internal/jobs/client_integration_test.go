package jobs_test

import (
	"context"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type RiverClientIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestRiverClientIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RiverClientIntegrationSuite))
}

// Pins the deferred worker registration app.go depends on: a job whose worker
// never registered would fail only at runtime.
func (s *RiverClientIntegrationSuite) TestDispatchedJobReachesItsWorker() {
	pool, err := pgxpool.New(s.Ctx, s.TC.DSN)
	s.Require().NoError(err)
	defer pool.Close()

	client, workers, err := jobs.NewClient(pool, zap.NewNop(), config.QueueConfig{
		Workers:        1,
		MaxAttempts:    5,
		PollIntervalMs: 1000,
		JobTimeoutSec:  900,
	}, "test", nil)
	s.Require().NoError(err)

	s.Require().NoError(jobs.RegisterWorkers(workers, s.TC.App, zap.NewNop()),
		"every job kind must register exactly once; queue and scheduler kinds must not collide")

	dispatcher, err := jobqueue.NewRiverDispatcher(client, otel.GetMeterProvider().Meter("test"))
	s.Require().NoError(err)

	runCtx, cancel := context.WithCancel(s.Ctx)
	defer cancel()
	s.Require().NoError(client.Start(runCtx))
	defer func() {
		stopCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = client.Stop(stopCtx)
	}()

	causer := int64(1)
	s.Require().NoError(dispatcher.Dispatch(s.Ctx, jobqueue.ActivityLogArgs{
		Event:    "create",
		Category: "river_probe",
		Causer:   &causer,
	}))

	s.Require().Eventually(func() bool {
		var n int64
		err := s.TC.DB.Table("activity_logs").Where("category = ?", "river_probe").Count(&n).Error
		return err == nil && n == 1
	}, 15*time.Second, 100*time.Millisecond, "the dispatched job was never worked")
}

// The dispatcher passes its own InsertOpts to carry the trace context; those must
// not blank out the queue the args ask for.
func (s *RiverClientIntegrationSuite) TestRebuildJobsLandOnTheRebuildQueue() {
	pool, err := pgxpool.New(s.Ctx, s.TC.DSN)
	s.Require().NoError(err)
	defer pool.Close()

	client, workers, err := jobs.NewClient(pool, zap.NewNop(), config.QueueConfig{
		Workers:        1,
		MaxAttempts:    5,
		PollIntervalMs: 1000,
		JobTimeoutSec:  900,
	}, "test", nil)
	s.Require().NoError(err)

	// River refuses to insert a kind with no worker. The client never starts, so
	// the jobs stay queued rather than rebuilding anything.
	s.Require().NoError(jobs.RegisterWorkers(workers, s.TC.App, zap.NewNop()))

	dispatcher, err := jobqueue.NewRiverDispatcher(client, otel.GetMeterProvider().Meter("test"))
	s.Require().NoError(err)

	for _, args := range []jobqueue.Job{
		jobqueue.BackfillAssetCashFlowsArgs{},
		jobqueue.CorrectFeeAccountingArgs{},
		jobqueue.MigrateZeroCostTradesArgs{},
	} {
		s.Require().NoError(dispatcher.Dispatch(s.Ctx, args))

		var gotQueue string
		s.Require().NoError(s.TC.DB.Table("river_job").
			Select("queue").
			Where("kind = ?", args.Kind()).
			Scan(&gotQueue).Error)
		s.Assert().Equal(jobqueue.QueueRebuild, gotQueue, "%s landed on the wrong queue", args.Kind())
	}
}
