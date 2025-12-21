package services

import (
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
)

type InvestmentServiceInterface interface {
}

type InvestmentService struct {
	repo          repositories.InvestmentRepositoryInterface
	accRepo       repositories.AccountRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
}

func NewInvestmentService(
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
) *InvestmentService {
	return &InvestmentService{
		repo:          repo,
		accRepo:       accRepo,
		jobDispatcher: jobDispatcher,
		loggingRepo:   loggingRepo,
	}
}

var _ InvestmentServiceInterface = (*InvestmentService)(nil)
