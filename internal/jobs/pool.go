package jobs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"go.uber.org/zap"
)

const (
	defaultWorkers    = 4
	maxDistinctErrors = 5
)

type poolFailure struct {
	err   error
	count int
}

type poolResult struct {
	total     int
	unit      string
	mu        sync.Mutex
	processed int
	failed    int
	order     []string
	byMsg     map[string]*poolFailure
}

func newPoolResult(total int, unit string) *poolResult {
	return &poolResult{total: total, unit: unit, byMsg: make(map[string]*poolFailure)}
}

func runPool[T any](ctx context.Context, items []T, workers int, unit string, fn func(context.Context, T) error) *poolResult {
	if workers < 1 {
		workers = defaultWorkers
	}

	res := newPoolResult(len(items), unit)

	queue := make(chan T, len(items))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range queue {
				select {
				case <-ctx.Done():
					return
				default:
				}
				res.record(fn(ctx, item))
			}
		}()
	}

	for _, item := range items {
		queue <- item
	}
	close(queue)
	wg.Wait()

	return res
}

func (r *poolResult) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.processed++
	if err == nil {
		return
	}
	r.failed++

	msg := err.Error()
	f, ok := r.byMsg[msg]
	if !ok {
		f = &poolFailure{err: err}
		r.byMsg[msg] = f
		r.order = append(r.order, msg)
	}
	f.count++
}

func (r *poolResult) logFailures(logger *zap.Logger, failMsg string) {
	if r.failed == 0 {
		return
	}

	failures := make([]*poolFailure, 0, len(r.order))
	for _, msg := range r.order {
		failures = append(failures, r.byMsg[msg])
	}
	sort.SliceStable(failures, func(i, j int) bool { return failures[i].count > failures[j].count })

	shown := failures
	if len(shown) > maxDistinctErrors {
		shown = shown[:maxDistinctErrors]
	}
	for _, f := range shown {
		logger.Error(failMsg, zap.Int("affected", f.count), zap.Error(f.err))
	}

	if len(failures) > len(shown) {
		hidden := 0
		for _, f := range failures[len(shown):] {
			hidden += f.count
		}
		logger.Error(failMsg,
			zap.Int("affected", hidden),
			zap.Int("distinct_errors_not_shown", len(failures)-len(shown)))
	}
}

func (r *poolResult) log(logger *zap.Logger, failMsg string) {
	r.logFailures(logger, failMsg)
	logger.Info("Completed",
		zap.Int("total", r.total),
		zap.Int("success", r.processed-r.failed),
		zap.Int("failed", r.failed),
		zap.Int("not_run", r.total-r.processed))
}

func (r *poolResult) stoppedErr(ctx context.Context) error {
	if r.processed >= r.total {
		return nil
	}
	cause := ctx.Err()
	if cause == nil {
		cause = errors.New("workers exited early")
	}
	return fmt.Errorf("stopped after %d of %d %s: %w", r.processed, r.total, r.unit, cause)
}

func (r *poolResult) err(ctx context.Context) error {
	if err := r.stoppedErr(ctx); err != nil {
		return err
	}
	if r.failed > 0 {
		return fmt.Errorf("%d of %d %s failed", r.failed, r.total, r.unit)
	}
	return nil
}
