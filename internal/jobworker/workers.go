package jobworker

import (
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/repositories"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// Must run before the client starts. Workers are long-lived and hold only deps;
// per-job data arrives as River args.
func RegisterWorkers(workers *river.Workers, c *bootstrap.ServiceContainer, logger *zap.Logger) error {
	// Infra, not domain services — built here rather than widening the container.
	loggingRepo := repositories.NewLoggingRepository(c.DB)
	notificationRepo := repositories.NewNotificationRepository(c.DB)
	analyticsRepo := repositories.NewAnalyticsRepository(c.DB)
	transactionRepo := repositories.NewTransactionRepository(c.DB)
	rebuildWorkers := c.Config.Scheduler.ConcurrentWorkers

	add := []func() error{
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewActivityLogWorker(loggingRepo))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewNotificationWorker(notificationRepo, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewRecalculateAssetPnLWorker(logger.Named("pnl_sync"), c.InvestmentService, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewSyncAssetAfterTradeWorker(logger.Named("asset_sync"), c.InvestmentService))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewRecalculateTemplateTimezoneWorker(logger.Named("template_tz"), transactionRepo))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewGenerateCategoryReportWorker(logger.Named("category_report"), analyticsRepo, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewBackfillAssetCashFlowsWorker(logger.Named("cashflow_backfill"), c.InvestmentService, rebuildWorkers))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewCorrectFeeAccountingWorker(logger.Named("fee_correction"), c.InvestmentService, rebuildWorkers))
		},
		func() error {
			return river.AddWorkerSafely(workers, queue_jobs.NewMigrateZeroCostTradesWorker(logger.Named("zero_cost_migration"), c.BackofficeService))
		},
	}

	for _, register := range add {
		if err := register(); err != nil {
			return err
		}
	}
	return nil
}
