package queue_jobs

import (
	"context"
	"sync"
	"wealth-warden/internal/joblog"

	"go.uber.org/zap"
)

const defaultUserWorkers = 4

type userPoolResult struct {
	processed int
	failures  *joblog.ErrorGroup
}

// runPerUser fans userIDs out over workers goroutines and stops handing out new
// users once ctx is done, so a cancelled run does not turn into one recorded
// failure per remaining user. processed reports how many actually ran.
func runPerUser(ctx context.Context, userIDs []int64, workers int, fn func(context.Context, int64) error) userPoolResult {
	if workers < 1 {
		workers = defaultUserWorkers
	}

	type result struct {
		userID int64
		err    error
	}

	jobs := make(chan int64, len(userIDs))
	results := make(chan result, len(userIDs))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uid := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				results <- result{userID: uid, err: fn(ctx, uid)}
			}
		}()
	}

	for _, uid := range userIDs {
		jobs <- uid
	}
	close(jobs)

	wg.Wait()
	close(results)

	out := userPoolResult{failures: joblog.NewErrorGroup("userID")}
	for r := range results {
		out.processed++
		out.failures.Add(r.userID, r.err)
	}
	return out
}

func (r userPoolResult) log(logger *zap.Logger, failMsg string, total int) {
	r.failures.Log(logger, failMsg)
	logger.Info("Completed",
		zap.Int("total", total),
		zap.Int("success", r.processed-r.failures.Count()),
		zap.Int("failed", r.failures.Count()),
		zap.Int("not_run", total-r.processed))
}
