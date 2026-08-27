package scheduler_jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/joblog"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
)

type AutomateTemplateJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	notifDispatcher   queue_jobs.NotificationDispatcher
	concurrentWorkers int
}

func NewAutomateTemplateJob(logger *zap.Logger, container *bootstrap.ServiceContainer, notifDispatcher queue_jobs.NotificationDispatcher, concurrentWorkers int) *AutomateTemplateJob {
	return &AutomateTemplateJob{
		logger:            logger,
		container:         container,
		notifDispatcher:   notifDispatcher,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AutomateTemplateJob) Run(ctx context.Context) error {
	templates, err := j.container.TransactionService.GetTemplatesReadyToRun(ctx, nil)
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

	type result struct {
		template *models.TransactionTemplate
		err      error
	}

	runPhase := func(phase []*models.TransactionTemplate, out chan<- result) {
		joblog.RunPool(ctx, phase, j.concurrentWorkers, func(ctx context.Context, tmpl *models.TransactionTemplate) {
			out <- result{template: tmpl, err: j.container.TransactionService.ProcessTemplate(ctx, tmpl)}
		})
	}

	results := make(chan result, len(templates))
	runPhase(inflows, results)
	runPhase(rest, results)
	close(results)

	type userSummary struct {
		succeeded []string
		failed    []string
	}

	successCount := 0
	alreadyRan := 0
	failures := joblog.NewErrorGroup("templateID")
	userResults := make(map[int64]*userSummary)

	for r := range results {
		if errors.Is(r.err, services.ErrTemplateAlreadyRanToday) {
			alreadyRan++
			continue
		}
		s, ok := userResults[r.template.UserID]
		if !ok {
			s = &userSummary{}
			userResults[r.template.UserID] = s
		}
		if r.err != nil {
			failures.Add(r.template.ID, r.err)
			s.failed = append(s.failed, r.template.Name)
		} else {
			s.succeeded = append(s.succeeded, r.template.Name)
			successCount++
		}
	}

	failures.Log(j.logger, "Failed to process template")

	j.logger.Info("Template processing completed",
		zap.Int("success", successCount),
		zap.Int("already_ran", alreadyRan),
		zap.Int("failed", failures.Count()))

	if j.notifDispatcher != nil {
		for userID, s := range userResults {
			if len(s.failed) > 0 {
				title := fmt.Sprintf("%d template(s) failed", len(s.failed))
				msg := strings.Join(s.failed, ",\n")
				_ = j.notifDispatcher.Dispatch(ctx, userID, title, msg, models.NotificationTypeError)
			}
			if len(s.succeeded) > 0 {
				title := fmt.Sprintf("%d template(s) executed", len(s.succeeded))
				msg := strings.Join(s.succeeded, ",\n")
				_ = j.notifDispatcher.Dispatch(ctx, userID, title, msg, models.NotificationTypeSuccess)
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("template processing stopped early: %w", err)
	}
	return nil
}
