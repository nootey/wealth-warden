package jobs

import (
	"context"
	"fmt"
	"sync"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type incomeFXBackfillSvc interface {
	BackfillIncomeExchangeRates(ctx context.Context, userID int64) (updated, skipped int, err error)
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type BackfillIncomeFXRatesWorker struct {
	river.WorkerDefaults[jobqueue.BackfillIncomeFXRatesArgs]
	logger            *zap.Logger
	workers           int
	investmentService incomeFXBackfillSvc
}

func NewBackfillIncomeFXRatesWorker(logger *zap.Logger, investmentService incomeFXBackfillSvc, workers int) *BackfillIncomeFXRatesWorker {
	if workers < 1 {
		workers = defaultWorkers
	}
	return &BackfillIncomeFXRatesWorker{
		logger:            logger,
		workers:           workers,
		investmentService: investmentService,
	}
}

func (w *BackfillIncomeFXRatesWorker) Work(ctx context.Context, _ *river.Job[jobqueue.BackfillIncomeFXRatesArgs]) error {
	userIDs, err := w.investmentService.GetUserIDsWithInvestments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		w.logger.Info("No users to process")
		return nil
	}

	w.logger.Info("Processing users", zap.Int("count", len(userIDs)))

	var (
		mu          sync.Mutex
		backfilled  int
		failed      int
		rowsUpdated int
		rowsSkipped int
	)

	g := new(errgroup.Group)
	g.SetLimit(w.workers)

	for _, userID := range userIDs {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			updated, skipped, err := w.investmentService.BackfillIncomeExchangeRates(ctx, userID)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				return fmt.Errorf("user %d: %w", userID, err)
			}
			backfilled++
			rowsUpdated += updated
			rowsSkipped += skipped
			return nil
		})
	}
	firstErr := g.Wait()

	w.logger.Info("Income exchange rate backfill completed",
		zap.Int("backfilled", backfilled),
		zap.Int("failed", failed),
		zap.Int("users_total", len(userIDs)),
		zap.Int("rows_updated", rowsUpdated),
		zap.Int("rows_skipped", rowsSkipped))

	if firstErr != nil {
		w.logger.Error("Failed to backfill income exchange rates",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("backfill stopped after %d of %d users: %w", backfilled+failed, len(userIDs), err)
	}
	if firstErr != nil {
		return fmt.Errorf("%d of %d users failed to backfill, first: %w", failed, len(userIDs), firstErr)
	}
	return nil
}
