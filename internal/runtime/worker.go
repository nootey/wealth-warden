package runtime

import "context"

type Worker interface {
	Name() string
	Start(ctx context.Context) error
	Shutdown() error
}

type worker struct {
	name     string
	start    func() error
	shutdown func() error
}

func NewWorker(name string, start func() error, shutdown func() error) Worker {
	return &worker{name: name, start: start, shutdown: shutdown}
}

func (w *worker) Name() string { return w.name }

func (w *worker) Start(ctx context.Context) error {
	if err := w.start(); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (w *worker) Shutdown() error { return w.shutdown() }
