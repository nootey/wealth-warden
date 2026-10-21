package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type importDeleter interface {
	RunImportDelete(ctx context.Context, userID, id int64) error
}

type ImportDeleteWorker struct {
	river.WorkerDefaults[jobqueue.ImportDeleteArgs]
	logger  *zap.Logger
	imports importDeleter
}

func NewImportDeleteWorker(logger *zap.Logger, imports importDeleter) *ImportDeleteWorker {
	return &ImportDeleteWorker{logger: logger, imports: imports}
}

func (w *ImportDeleteWorker) Work(ctx context.Context, job *river.Job[jobqueue.ImportDeleteArgs]) error {
	if err := w.imports.RunImportDelete(ctx, job.Args.UserID, job.Args.ImportID); err != nil {
		w.logger.Error("import delete failed", zap.Int64("import_id", job.Args.ImportID), zap.Error(err))
		return err
	}
	return nil
}
