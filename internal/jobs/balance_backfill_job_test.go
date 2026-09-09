package jobs_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type captureDispatcher struct {
	jobs []jobqueue.Job
}

func (d *captureDispatcher) Dispatch(_ context.Context, job jobqueue.Job) error {
	d.jobs = append(d.jobs, job)
	return nil
}

type fakeUserSvc struct {
	ids []int64
}

func (f fakeUserSvc) GetAllActiveUserIDs(context.Context) ([]int64, error) { return f.ids, nil }

type fakeAccountLister struct {
	ids []int64
}

func (f fakeAccountLister) ListOpenAccountIDs(_ context.Context, afterID int64, limit int) ([]int64, error) {
	var page []int64
	for _, id := range f.ids {
		if id > afterID {
			page = append(page, id)
		}
		if len(page) == limit {
			break
		}
	}
	return page, nil
}

type fakeAccountSvc struct {
	failFor int64
	seen    []int64
}

func (f *fakeAccountSvc) BackfillBalancesForUser(_ context.Context, userID int64, _, _ string) error {
	f.seen = append(f.seen, userID)
	if userID == f.failFor {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeAccountSvc) UpdateSnapshotMarketValues(context.Context, int64, time.Time) error {
	return nil
}

func seq(n int) []int64 {
	ids := make([]int64, n)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return ids
}

// Every user lands in exactly one batch, and the batch count is ceil(N / size).
func TestBalanceFanOutCoversEveryUserOnce(t *testing.T) {
	users := seq(250)
	dispatcher := &captureDispatcher{}

	job := jobs.NewBalanceBackfillJob(zap.NewNop(), fakeUserSvc{ids: users}, fakeAccountLister{}, dispatcher, 100, 500)
	require.NoError(t, job.Run(context.Background()))

	seen := map[int64]int{}
	batches := 0
	for _, j := range dispatcher.jobs {
		args, ok := j.(jobqueue.BalanceBackfillBatchArgs)
		if !ok {
			continue
		}
		batches++
		assert.LessOrEqual(t, len(args.UserIDs), 100)
		for _, id := range args.UserIDs {
			seen[id]++
		}
	}

	assert.Equal(t, 3, batches)
	assert.Len(t, seen, len(users))
	for _, id := range users {
		assert.Equal(t, 1, seen[id], "user %d was not in exactly one batch", id)
	}
}

// The reconcile pass is fanned out too, so no job carries the whole account population.
func TestBalanceFanOutPagesAccounts(t *testing.T) {
	dispatcher := &captureDispatcher{}

	job := jobs.NewBalanceBackfillJob(zap.NewNop(), fakeUserSvc{}, fakeAccountLister{ids: seq(12)}, dispatcher, 100, 5)
	require.NoError(t, job.Run(context.Background()))

	var pages [][]int64
	for _, j := range dispatcher.jobs {
		if args, ok := j.(jobqueue.BalanceReconcileBatchArgs); ok {
			pages = append(pages, args.AccountIDs)
		}
	}

	require.Len(t, pages, 3)
	assert.Equal(t, seq(5), pages[0])
	assert.Equal(t, []int64{11, 12}, pages[2])
}

// A broken user must not hide behind a batch that reports success, and it must not
// stop the other users in that batch.
func TestBalanceBackfillBatchFinishesTheBatchAndReportsFailures(t *testing.T) {
	accountSvc := &fakeAccountSvc{failFor: 3}
	worker := jobs.NewBalanceBackfillBatchWorker(zap.NewNop(), accountSvc)

	err := worker.Run(context.Background(), []int64{1, 2, 3, 4, 5})

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "1 of 5"), "error was %q", err)
	assert.Equal(t, []int64{1, 2, 3, 4, 5}, accountSvc.seen)
}
