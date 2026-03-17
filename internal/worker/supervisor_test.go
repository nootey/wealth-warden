package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
	"wealth-warden/internal/worker"

	"go.uber.org/zap"
)

func newTestSupervisor() *worker.Supervisor {
	return worker.NewSupervisor(zap.NewNop())
}

func TestSupervisor_RunsAllServices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count atomic.Int32
	s := newTestSupervisor()

	for range 3 {
		s.Add(worker.NewService("svc", func(ctx context.Context) {
			count.Add(1)
			<-ctx.Done()
		}))
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	s.Run(ctx)
	if count.Load() != 3 {
		t.Errorf("expected 3 services to run, got %d", count.Load())
	}
}

func TestSupervisor_WaitsForAllServicesToFinish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var done atomic.Int32
	s := newTestSupervisor()

	for range 3 {
		s.Add(worker.NewService("svc", func(ctx context.Context) {
			<-ctx.Done()
			time.Sleep(20 * time.Millisecond) // simulate cleanup
			done.Add(1)
		}))
	}

	cancel()
	s.Run(ctx)

	if done.Load() != 3 {
		t.Errorf("expected 3 services to finish, got %d", done.Load())
	}
}

func TestSupervisor_RecoversPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	s := newTestSupervisor()
	s.Add(worker.NewService("panicky", func(ctx context.Context) {
		panic("boom")
	}))

	cancel()

	// should not panic the test
	s.Run(ctx)
}

func TestSupervisor_RunsWithNoServices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := newTestSupervisor()
	s.Run(ctx) // should return immediately, no hang
}
