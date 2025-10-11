package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type ImportService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.ImportRepository
}

func NewImportService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ImportRepository,
) *ImportService {
	return &ImportService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
