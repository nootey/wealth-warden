package jobscheduler

import (
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobscheduler/scheduler_jobs"
	"wealth-warden/pkg/finance"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// Must run before the client starts.
func RegisterWorkers(workers *river.Workers, c *bootstrap.ServiceContainer, logger *zap.Logger) error {
	concurrentWorkers := c.Config.Scheduler.ConcurrentWorkers

	priceLogger := logger.Named(scheduler_jobs.TypeAssetPriceSync)
	priceClient, err := finance.NewPriceFetchClient(c.Config.FinanceAPIBaseURL)
	if err != nil {
		priceLogger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	historyLogger := logger.Named(scheduler_jobs.TypeAssetHistoryBackfill)
	balanceLogger := logger.Named(scheduler_jobs.TypeBalanceBackfill)
	recurringLogger := logger.Named(scheduler_jobs.TypeRecurringTransactions)

	add := []func() error{
		func() error {
			job := scheduler_jobs.NewAssetPriceHistoryBackfillJob(historyLogger, c.InvestmentService, c.DB, concurrentWorkers)
			return river.AddWorkerSafely(workers, scheduler_jobs.NewAssetPriceHistoryBackfillWorker(historyLogger, job))
		},
		func() error {
			job := scheduler_jobs.NewBalanceBackfillJob(balanceLogger, c, concurrentWorkers)
			return river.AddWorkerSafely(workers, scheduler_jobs.NewBalanceBackfillWorker(balanceLogger, job))
		},
		func() error {
			templates := scheduler_jobs.NewAutomateTemplateJob(recurringLogger.Named("templates"), c, c.NotifDispatcher, concurrentWorkers)
			goals := scheduler_jobs.NewAutoFundGoalsJob(recurringLogger.Named("savings_goals"), c, c.NotifDispatcher, concurrentWorkers)
			return river.AddWorkerSafely(workers, scheduler_jobs.NewRecurringTransactionsWorker(recurringLogger, templates.Run, goals.Run))
		},
		func() error {
			job := scheduler_jobs.NewAssetPriceSyncJob(priceLogger, c.InvestmentService, c.AccountService, c.DB, priceClient, c.NotifDispatcher, concurrentWorkers)
			return river.AddWorkerSafely(workers, scheduler_jobs.NewAssetPriceSyncWorker(priceLogger, job))
		},
	}

	for _, register := range add {
		if err := register(); err != nil {
			return err
		}
	}
	return nil
}
