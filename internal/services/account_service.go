package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type AccountService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.AccountRepository
}

func NewAccountService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.AccountRepository,
) *AccountService {
	return &AccountService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
