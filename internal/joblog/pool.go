package joblog

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

const defaultWorkers = 4

// Returns how many items actually ran, so a cancelled run is never recorded as a
// complete one. fn runs concurrently: whatever it writes to must be safe for that.
func RunPool[T any](ctx context.Context, items []T, workers int, fn func(context.Context, T)) int {
	// A count below one would start no goroutines at all, so the run would
	// report a clean pass having done nothing.
	if workers < 1 {
		workers = defaultWorkers
	}

	var processed atomic.Int64

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
				fn(ctx, item)
				processed.Add(1)
			}
		}()
	}

	for _, item := range items {
		queue <- item
	}
	close(queue)
	wg.Wait()

	return int(processed.Load())
}

type PoolResult struct {
	Processed int
	Failures  *ErrorGroup
}

func RunPerUser(ctx context.Context, userIDs []int64, workers int, fn func(context.Context, int64) error) PoolResult {
	failures := NewErrorGroup("userID")
	var mu sync.Mutex

	processed := RunPool(ctx, userIDs, workers, func(ctx context.Context, userID int64) {
		err := fn(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		failures.Add(userID, err)
	})

	return PoolResult{Processed: processed, Failures: failures}
}

func (r PoolResult) Log(logger *zap.Logger, failMsg string, total int) {
	r.Failures.Log(logger, failMsg)
	logger.Info("Completed",
		zap.Int("total", total),
		zap.Int("success", r.Processed-r.Failures.Count()),
		zap.Int("failed", r.Failures.Count()),
		zap.Int("not_run", total-r.Processed))
}
