package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type BalanceService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.BalanceRepository
}

func NewBalanceService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.BalanceRepository,
) *BalanceService {
	return &BalanceService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
