package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrQueueFull = errors.New("job queue is full")

// JobQueue handles background job processing using worker goroutines
type JobQueue struct {
	jobChannel  chan Job
	workerCount int
}

func NewJobQueue(workerCount int, queueSize int) *JobQueue {
	return &JobQueue{
		jobChannel:  make(chan Job, queueSize),
		workerCount: workerCount,
	}
}

func (q *JobQueue) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < q.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.worker(ctx)
		}()
	}
	wg.Wait()
}

func (q *JobQueue) Shutdown() error {
	close(q.jobChannel)
	return nil
}

func (q *JobQueue) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-q.jobChannel:
			if !ok {
				return
			}
			_ = job.Process(ctx)
		}
	}
}

func (q *JobQueue) AddJob(job Job) error {
	select {
	case q.jobChannel <- job:
		return nil
	default:
		return ErrQueueFull
	}
}
