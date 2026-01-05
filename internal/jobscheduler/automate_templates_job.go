package jobscheduler

import (
	"context"
	"fmt"
	"wealth-warden/internal/bootstrap"

	"go.uber.org/zap"
)

type AutomateTemplateJob struct {
	logger    *zap.Logger
	container *bootstrap.Container
}

func NewAutomateTemplateJob(logger *zap.Logger, container *bootstrap.Container) *AutomateTemplateJob {
	return &AutomateTemplateJob{
		logger:    logger,
		container: container,
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

	successCount := 0
	failCount := 0

	for _, template := range templates {
		if err := j.container.TransactionService.ProcessTemplate(ctx, template); err != nil {
			j.logger.Error("Failed to process template",
				zap.Int64("templateID", template.ID),
				zap.String("templateName", template.Name),
				zap.Error(err))
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
