package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Service struct {
	checkers []Checker
	states   map[string]error
	mu       sync.RWMutex
	gauge    metric.Float64Gauge
	logger   *zap.Logger
}

func New(logger *zap.Logger) (*Service, error) {
	gauge, err := otel.GetMeterProvider().Meter("wealth-warden").Float64Gauge(
		"health_check_up",
		metric.WithDescription("1 = healthy, 0 = unhealthy"),
	)
	if err != nil {
		return nil, err
	}
	return &Service{
		states: make(map[string]error),
		gauge:  gauge,
		logger: logger,
	}, nil
}

func (s *Service) Add(c Checker) {
	s.checkers = append(s.checkers, c)
}

func (s *Service) Run(ctx context.Context) {
	s.runChecks(ctx)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runChecks(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) runChecks(ctx context.Context) {
	for _, c := range s.checkers {
		err := c.Check(ctx)

		s.mu.Lock()
		s.states[c.Name()] = err
		s.mu.Unlock()

		val := 1.0
		if err != nil {
			val = 0.0
			s.logger.Warn("health check failed", zap.String("check", c.Name()), zap.Error(err))
		}
		s.gauge.Record(ctx, val, metric.WithAttributes(attribute.String("check", c.Name())))
	}
}

func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		states := make(map[string]error, len(s.states))
		for k, v := range s.states {
			states[k] = v
		}
		s.mu.RUnlock()

		type checkResult struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}

		checks := make(map[string]checkResult, len(states))
		allOK := true
		for name, err := range states {
			if err != nil {
				allOK = false
				checks[name] = checkResult{Status: "error", Error: err.Error()}
			} else {
				checks[name] = checkResult{Status: "ok"}
			}
		}

		overall := "ok"
		code := http.StatusOK
		if !allOK {
			overall = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": overall,
			"checks": checks,
		})
	})
}
