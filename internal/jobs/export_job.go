package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type exportRunner interface {
	RunExport(ctx context.Context, exportID, userID int64) error
}

type ExportWorker struct {
	river.WorkerDefaults[jobqueue.ExportArgs]
	logger  *zap.Logger
	exports exportRunner
}

func NewExportWorker(logger *zap.Logger, exports exportRunner) *ExportWorker {
	return &ExportWorker{logger: logger, exports: exports}
}

func (w *ExportWorker) Work(ctx context.Context, job *river.Job[jobqueue.ExportArgs]) error {
	if err := w.exports.RunExport(ctx, job.Args.ExportID, job.Args.UserID); err != nil {
		w.logger.Error("export run failed", zap.Int64("export_id", job.Args.ExportID), zap.Error(err))
		return err
	}
	return nil
}
