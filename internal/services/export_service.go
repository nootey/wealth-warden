package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type ExportService struct {
	Config     *config.Config
	Ctx        *DefaultServiceContext
	Repo       *repositories.ExportRepository
	TxnRepo    *repositories.TransactionRepository
	accService *AccountService
}

func NewExportService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ExportRepository,
	txnRepo *repositories.TransactionRepository,
	accService *AccountService,
) *ExportService {
	return &ExportService{
		Ctx:        ctx,
		Config:     cfg,
		Repo:       repo,
		TxnRepo:    txnRepo,
		accService: accService,
	}
}
