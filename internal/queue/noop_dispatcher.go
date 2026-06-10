package queue

import "context"

// NoopDispatcher drops dispatched jobs. Intended for seeders and tests where the
// async side effects (activity logs, etc.) are noise rather than work to run.
type NoopDispatcher struct{}

func (NoopDispatcher) Dispatch(context.Context, Job) error { return nil }
