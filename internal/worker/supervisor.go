package worker

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

type StartFn func(ctx context.Context)

type Service struct {
	name string
	run  StartFn
}

func NewService(name string, fn StartFn) *Service {
	return &Service{name: name, run: fn}
}

type Supervisor struct {
	logger   *zap.Logger
	services []*Service
	running  atomic.Int32
}

func NewSupervisor(logger *zap.Logger) *Supervisor {
	return &Supervisor{logger: logger}
}

func (s *Supervisor) Add(svc *Service) {
	s.services = append(s.services, svc)
}

func (s *Supervisor) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, svc := range s.services {
		svc := svc
		wg.Add(1)
		s.running.Add(1)
		go func() {
			defer wg.Done()
			defer s.running.Add(-1)
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("worker panicked", zap.String("service", svc.name), zap.Any("panic", r))
				}
			}()
			s.logger.Info("starting service", zap.String("service", svc.name))
			svc.run(ctx)
			s.logger.Info("service stopped", zap.String("service", svc.name))
		}()
	}

	<-ctx.Done()
	wg.Wait()
}
