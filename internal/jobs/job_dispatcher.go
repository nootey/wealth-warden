package jobs

type Job interface {
	Process()
}

type JobDispatcher interface {
	Dispatch(job Job) error
}
