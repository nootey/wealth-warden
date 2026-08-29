package jobs

import (
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

const defaultWorkers = 4

func RegisterWorkers(workers *river.Workers, c *bootstrap.ServiceContainer, logger *zap.Logger) error {
	loggingRepo := repositories.NewLoggingRepository(c.DB)
	notificationRepo := repositories.NewNotificationRepository(c.DB)
	concurrentWorkers := c.Config.Scheduler.ConcurrentWorkers

	priceLogger := logger.Named(jobqueue.TypeAssetPriceSync)
	priceClient, err := finance.NewPriceFetchClient(c.Config.FinanceAPIBaseURL)
	if err != nil {
		priceLogger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	historyLogger := logger.Named(jobqueue.TypeAssetHistoryBackfill)
	balanceLogger := logger.Named(jobqueue.TypeBalanceBackfill)
	recurringLogger := logger.Named(jobqueue.TypeRecurringTransactions)

	add := []func() error{
		func() error {
			return river.AddWorkerSafely(workers, NewActivityLogWorker(loggingRepo))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewNotificationWorker(notificationRepo, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewRecalculateAssetPnLWorker(logger.Named("pnl_sync"), c.InvestmentService, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewSyncAssetAfterTradeWorker(logger.Named("asset_sync"), c.InvestmentService))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewRecalculateTemplateTimezoneWorker(logger.Named("template_tz"), c.TransactionService))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewGenerateCategoryReportWorker(logger.Named("category_report"), c.AnalyticsService, c.Hub))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewBackfillAssetCashFlowsWorker(logger.Named("cashflow_backfill"), c.InvestmentService, concurrentWorkers))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewCorrectFeeAccountingWorker(logger.Named("fee_correction"), c.InvestmentService, concurrentWorkers))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewBackfillIncomeFXRatesWorker(logger.Named("income_fx_backfill"), c.InvestmentService, concurrentWorkers))
		},
		func() error {
			return river.AddWorkerSafely(workers, NewMigrateZeroCostTradesWorker(logger.Named("zero_cost_migration"), c.BackofficeService))
		},
		func() error {
			job := NewAssetPriceHistoryBackfillJob(historyLogger, c.InvestmentService, concurrentWorkers)
			return river.AddWorkerSafely(workers, NewAssetPriceHistoryBackfillWorker(historyLogger, job))
		},
		func() error {
			job := NewBalanceBackfillJob(balanceLogger, c.UserService, c.AccountService, concurrentWorkers)
			return river.AddWorkerSafely(workers, NewBalanceBackfillWorker(balanceLogger, job))
		},
		func() error {
			templates := NewAutomateTemplateJob(recurringLogger.Named("templates"), c.TransactionService, c.NotifDispatcher, concurrentWorkers)
			goals := NewAutoFundGoalsJob(recurringLogger.Named("savings_goals"), c.SavingsService, c.NotifDispatcher, concurrentWorkers)
			return river.AddWorkerSafely(workers, NewRecurringTransactionsWorker(recurringLogger, templates.Run, goals.Run))
		},
		func() error {
			job := NewAssetPriceSyncJob(priceLogger, c.InvestmentService, priceClient, c.NotifDispatcher, concurrentWorkers)
			return river.AddWorkerSafely(workers, NewAssetPriceSyncWorker(priceLogger, job))
		},
	}

	for _, register := range add {
		if err := register(); err != nil {
			return err
		}
	}
	return nil
}
