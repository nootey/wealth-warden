package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
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

func (s *InflowService) GetInflowsPaginated(paginationParams utils.PaginationParams) ([]models.Inflow, int, error) {

	totalRecords, err := s.InflowRepo.CountInflows()
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	inflows, err := s.InflowRepo.GetInflows(offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	return inflows, int(totalRecords), nil
}

func (s *InflowService) FetchAllInflowTypes() ([]models.InflowType, error) {
	return s.InflowRepo.GetAllInflowTypes()
}

func (s *InflowService) CreateInflow(inflow *models.Inflow) error {
	err := s.InflowRepo.SaveInflow(inflow)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) CreateInflowType(inflowType *models.InflowType) error {
	err := s.InflowRepo.SaveInflowType(inflowType)
	if err != nil {
		return err
	}
	return nil
}
