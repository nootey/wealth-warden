package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
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

func (s *InflowService) FetchAllInflowCategories() ([]models.InflowCategory, error) {
	return s.InflowRepo.GetAllInflowCategories()
}

func (s *InflowService) CreateInflow(inflow *models.Inflow) error {
	err := s.InflowRepo.SaveInflow(inflow)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) CreateInflowCategory(inflowCategory *models.InflowCategory) error {
	err := s.InflowRepo.SaveInflowCategory(inflowCategory)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) DeleteInflow(c *gin.Context, id uint) error {

	err := s.InflowRepo.DropInflow(id)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) DeleteInflowCategory(c *gin.Context, id uint) error {

	var count int64

	if err := s.InflowRepo.CountInflowsByCategory(id, &count); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete inflow category: it is being used by %d inflow(s)", count)
	}

	err := s.InflowRepo.DropInflowCategory(id)
	if err != nil {
		return err
	}
	return nil
}
