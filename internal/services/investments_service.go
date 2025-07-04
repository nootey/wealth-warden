package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type InvestmentsService struct {
	Config            *models.Config
	Ctx               *DefaultServiceContext
	InvestmentRepo    *repositories.InvestmentsRepository
	RecActionsService *ReoccurringActionService
}

func NewInvestmentsService(
	cfg *models.Config,
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
