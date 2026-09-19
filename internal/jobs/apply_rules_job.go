package jobs

import (
	"context"
	"wealth-warden/internal/jobqueue"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type ruleApplier interface {
	ApplyRules(ctx context.Context, userID int64) (scanned int, categorized int, err error)
}

type ApplyRulesWorker struct {
	river.WorkerDefaults[jobqueue.ApplyRulesArgs]
	logger *zap.Logger
	rules  ruleApplier
}

func NewApplyRulesWorker(logger *zap.Logger, rules ruleApplier) *ApplyRulesWorker {
	return &ApplyRulesWorker{logger: logger, rules: rules}
}

func (w *ApplyRulesWorker) Work(ctx context.Context, job *river.Job[jobqueue.ApplyRulesArgs]) error {
	scanned, categorized, err := w.rules.ApplyRules(ctx, job.Args.UserID)
	if err != nil {
		w.logger.Error("apply rules run failed", zap.Int64("user_id", job.Args.UserID), zap.Error(err))
		return err
	}

	// Best effort: the count is for the job history, not the run's success.
	if err := river.RecordOutput(ctx, map[string]int{"scanned": scanned, "categorized": categorized}); err != nil {
		w.logger.Warn("failed to record apply rules output", zap.Int64("user_id", job.Args.UserID), zap.Error(err))
	}

	w.logger.Info("Applied rules",
		zap.Int64("user_id", job.Args.UserID),
		zap.Int("scanned", scanned),
		zap.Int("categorized", categorized),
	)
	return nil
}
