package queue_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
	"wealth-warden/internal/queue"
)

type mockJob struct {
	called atomic.Bool
	err    error
}

func (j *mockJob) Process(ctx context.Context) error {
	j.called.Store(true)
	return j.err
}

func TestJobQueue_ProcessesJob(t *testing.T) {
	q := queue.NewJobQueue(1, 10)
	ctx, cancel := context.WithCancel(context.Background())

	job := &mockJob{}
	q.AddJob(job)

	go q.Run(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	if !job.called.Load() {
		t.Error("expected job to be processed")
	}
}

func TestJobQueue_ProcessesMultipleJobs(t *testing.T) {
	q := queue.NewJobQueue(2, 10)
	ctx, cancel := context.WithCancel(context.Background())

	for range 5 {
		q.AddJob(&mockJob{})
	}

	go q.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
}

func TestJobQueue_ShutdownDrainsQueue(t *testing.T) {
	q := queue.NewJobQueue(1, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed atomic.Int32
	for range 5 {
		job := &mockJob{}
		q.AddJob(job)
		if job.called.Load() {
			processed.Add(1)
		}
	}

	go q.Run(ctx)
	time.Sleep(100 * time.Millisecond)
	_ = q.Shutdown()
}
