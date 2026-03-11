package runtime

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

type Supervisor struct {
	logger  *zap.Logger
	workers []Worker
}

func NewSupervisor(logger *zap.Logger, workers ...Worker) *Supervisor {
	return &Supervisor{logger: logger, workers: workers}
}

func (s *Supervisor) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, len(s.workers))

	for _, w := range s.workers {
		w := w
		go func() {
			if err := w.Start(ctx); err != nil {
				s.logger.Error("Worker failed", zap.String("worker", w.Name()), zap.Error(err))
				errCh <- fmt.Errorf("worker %s: %w", w.Name(), err)
				stop()
			}
		}()
	}

	select {
	case err := <-errCh:
		s.logger.Error("Supervisor shutting down due to worker failure", zap.Error(err))
	case <-ctx.Done():
		s.logger.Info("Supervisor received shutdown signal")
	}

	return s.shutdown()
}

func (s *Supervisor) shutdown() error {
	var errs []error
	for _, w := range s.workers {
		if err := w.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("worker %s: %w", w.Name(), err))
		}
	}
	return errors.Join(errs...)
}
