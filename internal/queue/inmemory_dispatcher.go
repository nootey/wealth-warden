package queue

// InMemoryDispatcher is a simple dispatcher that pushes jobs to an in-memory queue
// This can be swapped out later for Redis, Kafka, etc.
type InMemoryDispatcher struct {
	Queue *JobQueue
}

func (d *InMemoryDispatcher) Dispatch(job Job) error {
	return d.Queue.AddJob(job)
}
