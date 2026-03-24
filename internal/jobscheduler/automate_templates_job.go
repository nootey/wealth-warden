package jobscheduler

import (
	"context"
	"fmt"
	"sync"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"

	"go.uber.org/zap"
)

type AutomateTemplateJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	concurrentWorkers int
}

func NewAutomateTemplateJob(logger *zap.Logger, container *bootstrap.ServiceContainer, concurrentWorkers int) *AutomateTemplateJob {
	return &AutomateTemplateJob{
		logger:            logger,
		container:         container,
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

	type result struct {
		template *models.TransactionTemplate
		err      error
	}

	jobs := make(chan *models.TransactionTemplate, len(templates))
	results := make(chan result, len(templates))

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
				results <- result{template: tmpl, err: err}
			}
		}()
	}

	for _, tmpl := range templates {
		jobs <- tmpl
	}
	close(jobs)

	wg.Wait()
	close(results)

	successCount := 0
	failCount := 0
	for r := range results {
		if r.err != nil {
			j.logger.Error("Failed to process template",
				zap.Int64("templateID", r.template.ID),
				zap.String("templateName", r.template.Name),
				zap.Error(r.err))
			failCount++
		} else {
			successCount++
		}
	}

	j.logger.Info("Template processing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))

	return nil
}
