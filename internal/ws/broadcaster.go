package ws

// Send must not block: a stalled connection cannot be allowed to wedge a job.
type Broadcaster interface {
	Send(userID int64, event Event)
}

type NoopBroadcaster struct{}

func (NoopBroadcaster) Send(int64, Event) {}
