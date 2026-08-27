package joblog_test

import (
	"context"
	"sync/atomic"
	"testing"
	"wealth-warden/internal/joblog"
)

func TestRunPool_RunsEveryItem(t *testing.T) {
	var ran atomic.Int64

	processed := joblog.RunPool(context.Background(), []int{1, 2, 3, 4, 5}, 2, func(context.Context, int) {
		ran.Add(1)
	})

	if processed != 5 {
		t.Errorf("processed = %d, want 5", processed)
	}
	if ran.Load() != 5 {
		t.Errorf("fn ran %d times, want 5", ran.Load())
	}
}

func TestRunPool_StopsOnCancelledContext(t *testing.T) {
	var ran atomic.Int64

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processed := joblog.RunPool(ctx, []int{1, 2, 3, 4, 5}, 2, func(context.Context, int) {
		ran.Add(1)
	})

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}
	if ran.Load() != 0 {
		t.Errorf("fn ran %d times on a cancelled context, want 0", ran.Load())
	}
}

func TestRunPool_ZeroWorkersStillRuns(t *testing.T) {
	processed := joblog.RunPool(context.Background(), []int{1, 2, 3}, 0, func(context.Context, int) {})

	if processed != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}
}

func TestRunPerUser_GroupsFailures(t *testing.T) {
	res := joblog.RunPerUser(context.Background(), []int64{1, 2, 3}, 2, func(_ context.Context, userID int64) error {
		if userID == 2 {
			return context.DeadlineExceeded
		}
		return nil
	})

	if res.Processed != 3 {
		t.Errorf("Processed = %d, want 3", res.Processed)
	}
	if res.Failures.Count() != 1 {
		t.Errorf("Failures.Count() = %d, want 1", res.Failures.Count())
	}
}
