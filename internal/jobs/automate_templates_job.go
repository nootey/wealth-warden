package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type templateResult struct {
	template *models.TransactionTemplate
	err      error
}

type userTemplateSummary struct {
	succeeded []string
	failed    []string
}

type AutomateTemplateJob struct {
	logger            *zap.Logger
	transactionSvc    services.TransactionServiceInterface
	notifDispatcher   jobqueue.NotificationDispatcher
	concurrentWorkers int
}

func NewAutomateTemplateJob(logger *zap.Logger, transactionSvc services.TransactionServiceInterface, notifDispatcher jobqueue.NotificationDispatcher, concurrentWorkers int) *AutomateTemplateJob {
	if concurrentWorkers < 1 {
		concurrentWorkers = defaultWorkers
	}
	return &AutomateTemplateJob{
		logger:            logger,
		transactionSvc:    transactionSvc,
		notifDispatcher:   notifDispatcher,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AutomateTemplateJob) Run(ctx context.Context) error {
	templates, err := j.transactionSvc.GetTemplatesReadyToRun(ctx)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		j.logger.Info("No templates ready to process")
		return nil
	}

	j.logger.Info("Processing templates", zap.Int("count", len(templates)))

	// Split into two phases: inflows first, then expenses and transfers.
	// This ensures funds are available before withdrawals on the same day.
	var inflows, rest []*models.TransactionTemplate
	for _, tmpl := range templates {
		if tmpl.TemplateType == "transaction" && tmpl.TransactionType != nil && *tmpl.TransactionType == "income" {
			inflows = append(inflows, tmpl)
		} else {
			rest = append(rest, tmpl)
		}
	}

	results := make(chan templateResult, len(templates))
	cancelled := j.runPhase(ctx, inflows, results)
	cancelled += j.runPhase(ctx, rest, results)
	close(results)

	successCount, alreadyRan, failed := 0, 0, 0
	userResults := make(map[int64]*userTemplateSummary)
	var firstErr error

	for r := range results {
		if errors.Is(r.err, models.ErrTemplateAlreadyRanToday) {
			alreadyRan++
			continue
		}
		s, ok := userResults[r.template.UserID]
		if !ok {
			s = &userTemplateSummary{}
			userResults[r.template.UserID] = s
		}
		if r.err != nil {
			s.failed = append(s.failed, r.template.Name)
			failed++
			if firstErr == nil {
				firstErr = fmt.Errorf("template %q: %w", r.template.Name, r.err)
			}
		} else {
			s.succeeded = append(s.succeeded, r.template.Name)
			successCount++
		}
	}

	j.logger.Info("Template processing completed",
		zap.Int("success", successCount),
		zap.Int("already_ran", alreadyRan),
		zap.Int("failed", failed),
		zap.Int("cancelled", cancelled),
		zap.Int("templates_total", len(templates)))

	if firstErr != nil {
		j.logger.Error("Template processing had failures",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	j.notifyUsers(ctx, userResults)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("template processing stopped early: %w", err)
	}
	return nil
}

// Returns how many templates the phase dropped because the run was cancelled.
// They produce no result, so without this the summary counts would not add up.
func (j *AutomateTemplateJob) runPhase(ctx context.Context, phase []*models.TransactionTemplate, results chan<- templateResult) int {
	var cancelled atomic.Int64

	g := new(errgroup.Group)
	g.SetLimit(j.concurrentWorkers)

	for _, tmpl := range phase {
		g.Go(func() error {
			if ctx.Err() != nil {
				cancelled.Add(1)
				return nil
			}
			results <- templateResult{template: tmpl, err: j.transactionSvc.ProcessTemplate(ctx, tmpl)}
			return nil
		})
	}
	_ = g.Wait()
	return int(cancelled.Load())
}

func (j *AutomateTemplateJob) notifyUsers(ctx context.Context, userResults map[int64]*userTemplateSummary) {
	if j.notifDispatcher == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)

	for userID, s := range userResults {
		if len(s.failed) > 0 {
			title := fmt.Sprintf("%d template(s) failed", len(s.failed))
			j.dispatch(ctx, userID, title, strings.Join(s.failed, ", "), models.NotificationTypeError)
		}
		if len(s.succeeded) > 0 {
			title := fmt.Sprintf("%d template(s) executed", len(s.succeeded))
			j.dispatch(ctx, userID, title, strings.Join(s.succeeded, ", "), models.NotificationTypeSuccess)
		}
	}
}

func (j *AutomateTemplateJob) dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) {
	if err := j.notifDispatcher.Dispatch(ctx, userID, title, message, notifType); err != nil {
		j.logger.Error("Failed to dispatch notification",
			zap.Int64("user_id", userID),
			zap.String("title", title),
			zap.Error(err))
	}
}
