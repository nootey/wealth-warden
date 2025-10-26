package services

import (
	"wealth-warden/internal/models"
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

func (s *ExportService) FetchExports(userID int64) ([]models.Export, error) {
	return s.Repo.FindExports(nil, userID)
}

func (s *ExportService) FetchExportsByExportType(userID int64, exportType string) ([]models.Export, error) {
	return s.Repo.FindExportsByExportType(nil, userID, exportType)
}

func (s *ExportService) CreateExport(userID int64) error {
	return nil
}
