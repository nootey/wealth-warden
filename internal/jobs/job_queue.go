package jobs

import (
	"context"
	"sync"
)

// JobQueue handles background job processing using worker goroutines
type JobQueue struct {
	jobChannel chan Job
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewJobQueue initializes a new in-memory job queue with workerCount concurrent workers
func NewJobQueue(workerCount int, queueSize int) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &JobQueue{
		jobChannel: make(chan Job, queueSize),
		ctx:        ctx,
		cancel:     cancel,
	}

	for i := 0; i < workerCount; i++ {
		go q.worker()
	}

	return q
}

func (q *JobQueue) worker() {
	for {
		select {
		case <-q.ctx.Done():
			return
		case job, ok := <-q.jobChannel:
			if !ok {
				return
			}
			_ = job.Process(q.ctx)
			q.wg.Done()
		}
	}
}

func (q *JobQueue) AddJob(job Job) {
	q.wg.Add(1)
	q.jobChannel <- job
}

func (q *JobQueue) Shutdown() {
	q.cancel()
	close(q.jobChannel)
	q.wg.Wait()
}
