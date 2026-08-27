package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type zeroCostMigrationSvc interface {
	RunZeroCostTradeMigration(ctx context.Context) (*models.ZeroCostMigrationResult, error)
}

type MigrateZeroCostTradesWorker struct {
	river.WorkerDefaults[jobqueue.MigrateZeroCostTradesArgs]
	logger            *zap.Logger
	backofficeService zeroCostMigrationSvc
}

func NewMigrateZeroCostTradesWorker(logger *zap.Logger, backofficeService zeroCostMigrationSvc) *MigrateZeroCostTradesWorker {
	return &MigrateZeroCostTradesWorker{logger: logger, backofficeService: backofficeService}
}

func (w *MigrateZeroCostTradesWorker) Work(ctx context.Context, _ *river.Job[jobqueue.MigrateZeroCostTradesArgs]) error {
	result, err := w.backofficeService.RunZeroCostTradeMigration(ctx)
	if err != nil {
		return err
	}

	// The endpoint answers 202, so this log line is the only report of the run.
	w.logger.Info("Zero-cost trade migration completed",
		zap.Int("trades_processed", result.TotalProcessed),
		zap.Int("assets_processed", result.AssetsProcessed),
		zap.Int("assets_failed", result.AssetsFailed))

	return nil
}
