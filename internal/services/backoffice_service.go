package services

import (
	"context"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
)

type BackofficeServiceInterface interface {
	BackfillAssetCashFlows(ctx context.Context) error
}

type BackofficeService struct {
	jobDispatcher     queue.JobDispatcher
	repo              repositories.BackofficeRepositoryInterface
	investmentService InvestmentServiceInterface
	accountService    AccountServiceInterface
	userService       UserServiceInterface
}

func NewBackofficeService(
	jobDispatcher queue.JobDispatcher,
	repo *repositories.BackofficeRepository,
	investmentService InvestmentServiceInterface,
	accountService AccountServiceInterface,
	userService UserServiceInterface,
) *BackofficeService {
	return &BackofficeService{
		jobDispatcher:     jobDispatcher,
		repo:              repo,
		investmentService: investmentService,
		accountService:    accountService,
		userService:       userService,
	}
}

var _ BackofficeServiceInterface = (*BackofficeService)(nil)

func (s *BackofficeService) BackfillAssetCashFlows(ctx context.Context) error {
	return s.jobDispatcher.Dispatch(&queue.BackfillAssetCashFlowsJob{
		InvestmentService: s.investmentService,
		AccountService:    s.accountService,
		UserService:       s.userService,
	})
}
