package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type InvestmentsService struct {
	Config            *config.Config
	Ctx               *DefaultServiceContext
	InvestmentRepo    *repositories.InvestmentsRepository
	RecActionsService *ReoccurringActionService
}

func NewInvestmentsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.InvestmentsRepository,
	recActionsService *ReoccurringActionService,
) *InvestmentsService {
	return &InvestmentsService{
		Ctx:               ctx,
		Config:            cfg,
		InvestmentRepo:    repo,
		RecActionsService: recActionsService,
	}
}
