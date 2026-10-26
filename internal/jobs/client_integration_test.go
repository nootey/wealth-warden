//go:build integration

package jobs_test

import (
	"context"
	"testing"
	"time"
	"wealth-warden/internal/config"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/tests"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type RiverClientIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationRiverClientSuite(t *testing.T) {
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

type slowProbeArgs struct{}

func (slowProbeArgs) Kind() string { return "slow_probe" }

type slowProbeWorker struct {
	river.WorkerDefaults[slowProbeArgs]
	started chan struct{}
}

func (w *slowProbeWorker) Work(ctx context.Context, _ *river.Job[slowProbeArgs]) error {
	close(w.started)
	select {
	case <-time.After(2 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown cancels the context the service runs on. A job running at that moment
// must finish, not be cancelled: on its last attempt River would discard it.
func (s *RiverClientIntegrationSuite) TestShutdownLetsRunningJobFinish() {
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

	worker := &slowProbeWorker{started: make(chan struct{})}
	river.AddWorker(workers, worker)

	runCtx, cancel := context.WithCancel(s.Ctx)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		jobs.NewService(client, zap.NewNop()).Run(runCtx)
		close(stopped)
	}()

	res, err := client.Insert(s.Ctx, slowProbeArgs{}, &river.InsertOpts{MaxAttempts: 1})
	s.Require().NoError(err)

	select {
	case <-worker.started:
	case <-time.After(15 * time.Second):
		s.FailNow("the job never started")
	}

	cancel()

	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		s.FailNow("the service never stopped")
	}

	job, err := client.JobGet(s.Ctx, res.Job.ID)
	s.Require().NoError(err)
	s.Equal(rivertype.JobStateCompleted, job.State)
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
		s.Equal(jobqueue.QueueRebuild, gotQueue, "%s landed on the wrong queue", args.Kind())
	}
}
