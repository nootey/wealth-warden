package jobworker

import (
	"encoding/json"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

// jobFactory rebuilds a job from its serialized data, re-attaching live deps.
type jobFactory func(data []byte) (queue.Job, error)

// registry maps a stored job type to the factory that reconstructs it. Adding a
// new job type means adding one entry here keyed by its queue.Type* constant.
type registry struct {
	factories map[string]jobFactory
}

func newRegistry(c *bootstrap.ServiceContainer, logger *zap.Logger) *registry {
	// Repos the jobs hold are infra, not domain services — build them straight
	// from the shared DB rather than widening the service container.
	loggingRepo := repositories.NewLoggingRepository(c.DB)
	notificationRepo := repositories.NewNotificationRepository(c.DB)
	analyticsRepo := repositories.NewAnalyticsRepository(c.DB)
	transactionRepo := repositories.NewTransactionRepository(c.DB)

	factories := map[string]jobFactory{
		// Struct-literal jobs: exported deps re-attached after unmarshal.
		queue_jobs.TypeActivityLog: func(data []byte) (queue.Job, error) {
			var j queue_jobs.ActivityLogJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			j.LoggingRepo = loggingRepo
			return &j, nil
		},
		queue_jobs.TypeNotification: func(data []byte) (queue.Job, error) {
			var j queue_jobs.NotificationJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			j.Repo = notificationRepo
			j.Broadcaster = c.Hub
			return &j, nil
		},

		// Constructor jobs: unmarshal data fields, then rebuild via the public
		// constructor (deps are unexported, so they can only be set there).
		queue_jobs.TypeRecalculateAssetPnL: func(data []byte) (queue.Job, error) {
			var j queue_jobs.RecalculateAssetPnLJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue_jobs.NewRecalculateAssetPnLJob(logger.Named("pnl_sync"), c.InvestmentService, j.UserID, j.AssetID, j.AccountID), nil
		},
		queue_jobs.TypeSyncAssetAfterTrade: func(data []byte) (queue.Job, error) {
			var j queue_jobs.SyncAssetAfterTradeJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue_jobs.NewSyncAssetAfterTradeJob(logger.Named("asset_sync"), c.InvestmentService, j.UserID, j.AssetID, j.Ticker, j.InvestmentType, j.TradeDate), nil
		},
		queue_jobs.TypeRecalculateTemplateTZ: func(data []byte) (queue.Job, error) {
			var j queue_jobs.RecalculateTemplateTimezoneJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue_jobs.NewRecalculateTemplateTimezoneJob(logger.Named("template_tz"), transactionRepo, j.UserID, j.OldTimezone, j.NewTimezone), nil
		},
		queue_jobs.TypeGenerateCategoryReport: func(data []byte) (queue.Job, error) {
			var j queue_jobs.GenerateCategoryReportJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue_jobs.NewGenerateCategoryReportJob(logger.Named("category_report"), analyticsRepo, c.Hub, j.ReportID, j.UserID, j.Params), nil
		},

		// Payload-less maintenance jobs: deps only.
		queue_jobs.TypeBackfillAssetCashFlows: func([]byte) (queue.Job, error) {
			return queue_jobs.NewBackfillAssetCashFlowsJob(logger.Named("cashflow_backfill"), c.InvestmentService, c.AccountService, c.UserService), nil
		},
		queue_jobs.TypeCorrectFeeAccounting: func([]byte) (queue.Job, error) {
			return queue_jobs.NewCorrectFeeAccountingJob(logger.Named("fee_correction"), c.InvestmentService, c.AccountService, c.UserService), nil
		},
	}

	return &registry{factories: factories}
}

func (r *registry) build(jobType string, data []byte) (queue.Job, error) {
	f, ok := r.factories[jobType]
	if !ok {
		return nil, fmt.Errorf("unknown job type %q", jobType)
	}
	return f(data)
}
