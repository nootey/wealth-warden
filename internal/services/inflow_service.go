package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type InflowService struct {
	InflowRepo *repositories.InflowRepository
	Config     *config.Config
}

func NewInflowService(cfg *config.Config, repo *repositories.InflowRepository) *InflowService {
	return &InflowService{
		InflowRepo: repo,
		Config:     cfg,
	}
}

func (s *InflowService) FetchAllInflowTypes() ([]models.InflowType, error) {
	return s.InflowRepo.GetAllInflowTypes()
}

func (s *InflowService) CreateInflowType(inflowType *models.InflowType) error {
	err := s.InflowRepo.SaveInflowType(inflowType)
	if err != nil {
		return err
	}
	return nil
}
