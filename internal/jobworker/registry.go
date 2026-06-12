package jobworker

import (
	"encoding/json"
	"fmt"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/queue"
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
		queue.TypeActivityLog: func(data []byte) (queue.Job, error) {
			var j queue.ActivityLogJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			j.LoggingRepo = loggingRepo
			return &j, nil
		},
		queue.TypeNotification: func(data []byte) (queue.Job, error) {
			var j queue.NotificationJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			j.Repo = notificationRepo
			return &j, nil
		},

		// Constructor jobs: unmarshal data fields, then rebuild via the public
		// constructor (deps are unexported, so they can only be set there).
		queue.TypeRecalculateAssetPnL: func(data []byte) (queue.Job, error) {
			var j queue.RecalculateAssetPnLJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue.NewRecalculateAssetPnLJob(logger.Named("pnl_sync"), c.InvestmentService, j.UserID, j.AssetID, j.AccountID), nil
		},
		queue.TypeSyncAssetAfterTrade: func(data []byte) (queue.Job, error) {
			var j queue.SyncAssetAfterTradeJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue.NewSyncAssetAfterTradeJob(logger.Named("asset_sync"), c.InvestmentService, j.UserID, j.AssetID, j.Ticker, j.InvestmentType, j.TradeDate), nil
		},
		queue.TypeRecalculateTemplateTZ: func(data []byte) (queue.Job, error) {
			var j queue.RecalculateTemplateTimezoneJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue.NewRecalculateTemplateTimezoneJob(logger.Named("template_tz"), transactionRepo, j.UserID, j.OldTimezone, j.NewTimezone), nil
		},
		queue.TypeGenerateCategoryReport: func(data []byte) (queue.Job, error) {
			var j queue.GenerateCategoryReportJob
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return queue.NewGenerateCategoryReportJob(logger.Named("category_report"), analyticsRepo, j.ReportID, j.UserID, j.Params), nil
		},

		// Payload-less maintenance jobs: deps only.
		queue.TypeBackfillAssetCashFlows: func([]byte) (queue.Job, error) {
			return queue.NewBackfillAssetCashFlowsJob(logger.Named("cashflow_backfill"), c.InvestmentService, c.AccountService, c.UserService), nil
		},
		queue.TypeCorrectFeeAccounting: func([]byte) (queue.Job, error) {
			return queue.NewCorrectFeeAccountingJob(logger.Named("fee_correction"), c.InvestmentService, c.AccountService, c.UserService), nil
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
