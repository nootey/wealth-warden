package jobscheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"

	"go.uber.org/zap"
)

type AutomateTemplateJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	notifDispatcher   queue.NotificationDispatcher
	concurrentWorkers int
}

func NewAutomateTemplateJob(logger *zap.Logger, container *bootstrap.ServiceContainer, notifDispatcher queue.NotificationDispatcher, concurrentWorkers int) *AutomateTemplateJob {
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
		if len(phase) == 0 {
			return
		}
		jobs := make(chan *models.TransactionTemplate, len(phase))
		var wg sync.WaitGroup
		for i := 0; i < j.concurrentWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for tmpl := range jobs {
					select {
					case <-ctx.Done():
						return
					default:
					}
					err := j.container.TransactionService.ProcessTemplate(ctx, tmpl)
					out <- result{template: tmpl, err: err}
				}
			}()
		}
		for _, tmpl := range phase {
			jobs <- tmpl
		}
		close(jobs)
		wg.Wait()
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
	failCount := 0
	userResults := make(map[int64]*userSummary)

	for r := range results {
		s, ok := userResults[r.template.UserID]
		if !ok {
			s = &userSummary{}
			userResults[r.template.UserID] = s
		}
		if r.err != nil {
			j.logger.Error("Failed to process template",
				zap.Int64("templateID", r.template.ID),
				zap.String("templateName", r.template.Name),
				zap.Error(r.err))
			s.failed = append(s.failed, r.template.Name)
			failCount++
		} else {
			s.succeeded = append(s.succeeded, r.template.Name)
			successCount++
		}
	}

	j.logger.Info("Template processing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	if j.notifDispatcher != nil {
		for userID, s := range userResults {
			if len(s.failed) > 0 {
				title := fmt.Sprintf("%d template(s) failed", len(s.failed))
				msg := strings.Join(s.failed, ",\n")
				_ = j.notifDispatcher.Dispatch(userID, title, msg, models.NotificationTypeError)
			}
			if len(s.succeeded) > 0 {
				title := fmt.Sprintf("%d template(s) executed", len(s.succeeded))
				msg := strings.Join(s.succeeded, ",\n")
				_ = j.notifDispatcher.Dispatch(userID, title, msg, models.NotificationTypeSuccess)
			}
		}
	}

	return nil
}
