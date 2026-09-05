package jobs

import (
	"context"
	"encoding/json"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/ws"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// River writes a job's final state only after the worker returns, so a worker
// cannot broadcast it. These events fire after the write.
type UserJobNotifier struct {
	events      <-chan *river.Event
	cancel      func()
	broadcaster ws.Broadcaster
	logger      *zap.Logger
}

func NewUserJobNotifier(client *river.Client[pgx.Tx], broadcaster ws.Broadcaster, logger *zap.Logger) *UserJobNotifier {
	events, cancel := client.Subscribe(
		river.EventKindJobCompleted,
		river.EventKindJobFailed,
		river.EventKindJobCancelled,
	)
	return &UserJobNotifier{events: events, cancel: cancel, broadcaster: broadcaster, logger: logger}
}

func (n *UserJobNotifier) Run(ctx context.Context) {
	defer n.cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-n.events:
			if !ok {
				return
			}
			n.forward(event)
		}
	}
}

func (n *UserJobNotifier) forward(event *river.Event) {
	if event.Job == nil || !jobqueue.SelfServiceKinds[event.Job.Kind] {
		return
	}

	var args struct {
		UserID int64
	}
	if err := json.Unmarshal(event.Job.EncodedArgs, &args); err != nil {
		n.logger.Warn("failed to decode user job args",
			zap.String("kind", event.Job.Kind), zap.Int64("job_id", event.Job.ID), zap.Error(err))
		return
	}
	if args.UserID == 0 {
		return
	}

	n.broadcaster.Send(args.UserID, ws.Event{
		Type:    ws.TypeUserJobUpdated,
		Payload: ws.UserJobPayload{Kind: event.Job.Kind, State: string(event.Job.State)},
	})
}
