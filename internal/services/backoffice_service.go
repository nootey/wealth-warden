package services

import (
	"context"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

type BackofficeServiceInterface interface {
	BackfillAssetCashFlows(ctx context.Context) error
}

type BackofficeService struct {
	logger            *zap.Logger
	jobDispatcher     queue.JobDispatcher
	repo              repositories.BackofficeRepositoryInterface
	investmentService InvestmentServiceInterface
	accountService    AccountServiceInterface
	userService       UserServiceInterface
}

func NewBackofficeService(
	logger *zap.Logger,
	jobDispatcher queue.JobDispatcher,
	repo *repositories.BackofficeRepository,
	investmentService InvestmentServiceInterface,
	accountService AccountServiceInterface,
	userService UserServiceInterface,
) *BackofficeService {
	return &BackofficeService{
		logger:            logger,
		jobDispatcher:     jobDispatcher,
		repo:              repo,
		investmentService: investmentService,
		accountService:    accountService,
		userService:       userService,
	}
}

var _ BackofficeServiceInterface = (*BackofficeService)(nil)

func (s *BackofficeService) BackfillAssetCashFlows(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(queue.NewBackfillAssetCashFlowsJob(
		s.logger.Named("backfill_asset_cash_flows"),
		s.investmentService,
		s.accountService,
		s.userService,
	))
}
