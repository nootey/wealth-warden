package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRunPoolProcessesEveryItem(t *testing.T) {
	var calls atomic.Int64

	res := runPool(context.Background(), []int{1, 2, 3, 4, 5}, 2, "items", func(context.Context, int) error {
		calls.Add(1)
		return nil
	})

	if calls.Load() != 5 {
		t.Fatalf("called %d times, want 5", calls.Load())
	}
	if res.processed != 5 || res.failed != 0 {
		t.Fatalf("processed=%d failed=%d, want 5 and 0", res.processed, res.failed)
	}
	if err := res.err(context.Background()); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestRunPoolDefaultsWorkerCount(t *testing.T) {
	res := runPool(context.Background(), []int{1, 2, 3}, 0, "items", func(context.Context, int) error { return nil })

	if res.processed != 3 {
		t.Fatalf("processed = %d, want 3", res.processed)
	}
}

func TestRunPoolStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := runPool(ctx, []int{1, 2, 3, 4, 5}, 2, "items", func(context.Context, int) error { return nil })

	if res.processed != 0 {
		t.Fatalf("processed = %d, want 0", res.processed)
	}
	err := res.stoppedErr(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stoppedErr = %v, want context.Canceled", err)
	}
}

func TestPoolResultErrReportsFailures(t *testing.T) {
	res := runPool(context.Background(), []int{1, 2, 3}, 2, "users", func(_ context.Context, i int) error {
		if i == 2 {
			return errors.New("boom")
		}
		return nil
	})

	if res.failed != 1 {
		t.Fatalf("failed = %d, want 1", res.failed)
	}
	if got := res.err(context.Background()).Error(); got != "1 of 3 users failed" {
		t.Fatalf("err = %q", got)
	}
}

func TestLogFailuresGroupsByMessage(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)

	res := newPoolResult(30, "items")
	for i := 0; i < 20; i++ {
		res.record(errors.New("db down"))
	}
	for i := 0; i < 10; i++ {
		res.record(errors.New("bad input"))
	}
	res.record(nil)
	res.logFailures(zap.New(core), "failed")

	if logs.Len() != 2 {
		t.Fatalf("logged %d lines, want 2", logs.Len())
	}
	if got := logs.All()[0].ContextMap()["affected"]; got != int64(20) {
		t.Fatalf("first line affected = %v, want 20 (most common first)", got)
	}
	if res.failed != 30 {
		t.Fatalf("failed = %d, want 30", res.failed)
	}
}

func TestLogFailuresCapsDistinctErrors(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)

	res := newPoolResult(7, "items")
	for i := 0; i < 7; i++ {
		res.record(fmt.Errorf("error %d", i))
	}
	res.logFailures(zap.New(core), "failed")

	if logs.Len() != maxDistinctErrors+1 {
		t.Fatalf("logged %d lines, want %d", logs.Len(), maxDistinctErrors+1)
	}
	last := logs.All()[logs.Len()-1].ContextMap()
	if last["distinct_errors_not_shown"] != int64(2) || last["affected"] != int64(2) {
		t.Fatalf("overflow line = %v", last)
	}
}
