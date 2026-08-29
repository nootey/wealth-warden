package jobqueue

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type JobManager interface {
	JobGet(ctx context.Context, id int64) (*rivertype.JobRow, error)
	JobRetry(ctx context.Context, id int64) (*rivertype.JobRow, error)
	JobCancel(ctx context.Context, id int64) (*rivertype.JobRow, error)
	JobDelete(ctx context.Context, id int64) (*rivertype.JobRow, error)
	QueueList(ctx context.Context) ([]*rivertype.Queue, error)
	QueuePause(ctx context.Context, name string) error
	QueueResume(ctx context.Context, name string) error
}

type NoopJobManager struct{}

func (NoopJobManager) JobGet(context.Context, int64) (*rivertype.JobRow, error)    { return nil, nil }
func (NoopJobManager) JobRetry(context.Context, int64) (*rivertype.JobRow, error)  { return nil, nil }
func (NoopJobManager) JobCancel(context.Context, int64) (*rivertype.JobRow, error) { return nil, nil }
func (NoopJobManager) JobDelete(context.Context, int64) (*rivertype.JobRow, error) { return nil, nil }
func (NoopJobManager) QueueList(context.Context) ([]*rivertype.Queue, error)       { return nil, nil }
func (NoopJobManager) QueuePause(context.Context, string) error                    { return nil }
func (NoopJobManager) QueueResume(context.Context, string) error                   { return nil }

type RiverJobManager struct {
	client *river.Client[pgx.Tx]
}

func NewRiverJobManager(client *river.Client[pgx.Tx]) *RiverJobManager {
	return &RiverJobManager{client: client}
}

var _ JobManager = (*RiverJobManager)(nil)

func (m *RiverJobManager) JobGet(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return m.client.JobGet(ctx, id)
}

func (m *RiverJobManager) JobRetry(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return m.client.JobRetry(ctx, id)
}

func (m *RiverJobManager) JobCancel(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return m.client.JobCancel(ctx, id)
}

func (m *RiverJobManager) JobDelete(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return m.client.JobDelete(ctx, id)
}

func (m *RiverJobManager) QueueList(ctx context.Context) ([]*rivertype.Queue, error) {
	res, err := m.client.QueueList(ctx, river.NewQueueListParams().First(100))
	if err != nil {
		return nil, err
	}
	return res.Queues, nil
}

func (m *RiverJobManager) QueuePause(ctx context.Context, name string) error {
	return m.client.QueuePause(ctx, name, nil)
}

func (m *RiverJobManager) QueueResume(ctx context.Context, name string) error {
	return m.client.QueueResume(ctx, name, nil)
}
