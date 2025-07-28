package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type CategoryService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.CategoryRepository
}

func NewCategoryService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.CategoryRepository,
) *CategoryService {
	return &CategoryService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
