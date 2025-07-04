package jobs

import (
	"sync"
)

// JobQueue handles background job processing using worker goroutines
type JobQueue struct {
	JobChannel chan Job
	wg         sync.WaitGroup
}

// NewJobQueue initializes a new in-memory job queue with workerCount concurrent workers
func NewJobQueue(workerCount int, queueSize int) *JobQueue {
	q := &JobQueue{
		JobChannel: make(chan Job, queueSize),
	}

	for i := 0; i < workerCount; i++ {
		go q.worker()
	}

	return q
}

func (q *JobQueue) worker() {
	for job := range q.JobChannel {
		job.Process()
		q.wg.Done()
	}
}

func (q *JobQueue) AddJob(job Job) {
	q.wg.Add(1)
	q.JobChannel <- job
}

func (q *JobQueue) Shutdown() {
	close(q.JobChannel)
	q.wg.Wait()
}
