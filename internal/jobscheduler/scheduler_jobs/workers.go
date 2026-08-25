package scheduler_jobs

const defaultWorkers = 4

// A count of zero would start no workers at all, so the job would report a
// clean run having done nothing.
func workerCount(n int) int {
	if n < 1 {
		return defaultWorkers
	}
	return n
}
