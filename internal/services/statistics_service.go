package services

import (
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type StatisticsService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.StatisticsRepository
}

func NewStatisticsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.StatisticsRepository,
) *StatisticsService {
	return &StatisticsService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
