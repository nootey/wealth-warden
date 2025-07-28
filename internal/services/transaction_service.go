package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type TransactionService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.TransactionRepository
}

func NewTransactionService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.TransactionRepository,
) *TransactionService {
	return &TransactionService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
