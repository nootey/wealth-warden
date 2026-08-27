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

type feeAccountingCorrectionSvc interface {
	CorrectFeeAccountingAndRebuild(ctx context.Context, userID int64) error
	GetUserIDsWithInvestments(ctx context.Context) ([]int64, error)
}

type CorrectFeeAccountingWorker struct {
	river.WorkerDefaults[jobqueue.CorrectFeeAccountingArgs]
	logger            *zap.Logger
	workers           int
	investmentService feeAccountingCorrectionSvc
}

func NewCorrectFeeAccountingWorker(logger *zap.Logger, investmentService feeAccountingCorrectionSvc, workers int) *CorrectFeeAccountingWorker {
	if workers < 1 {
		workers = defaultWorkers
	}
	return &CorrectFeeAccountingWorker{
		logger:            logger,
		workers:           workers,
		investmentService: investmentService,
	}
}

func (w *CorrectFeeAccountingWorker) Work(ctx context.Context, _ *river.Job[jobqueue.CorrectFeeAccountingArgs]) error {
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
		mu        sync.Mutex
		corrected int
		failed    int
	)

	g := new(errgroup.Group)
	g.SetLimit(w.workers)

	for _, userID := range userIDs {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			err := w.investmentService.CorrectFeeAccountingAndRebuild(ctx, userID)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				return fmt.Errorf("user %d: %w", userID, err)
			}
			corrected++
			return nil
		})
	}
	firstErr := g.Wait()

	w.logger.Info("Fee accounting correction completed",
		zap.Int("corrected", corrected),
		zap.Int("failed", failed),
		zap.Int("users_total", len(userIDs)))

	if firstErr != nil {
		w.logger.Error("Failed to correct fee accounting",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("correction stopped after %d of %d users: %w", corrected+failed, len(userIDs), err)
	}
	if firstErr != nil {
		return fmt.Errorf("%d of %d users failed to correct, first: %w", failed, len(userIDs), firstErr)
	}
	return nil
}
