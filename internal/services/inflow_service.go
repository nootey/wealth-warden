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
	InflowRepo  *repositories.InflowRepository
	AuthService *AuthService
	Config      *config.Config
}

func NewInflowService(cfg *config.Config, authService *AuthService, repo *repositories.InflowRepository) *InflowService {
	return &InflowService{
		InflowRepo:  repo,
		AuthService: authService,
		Config:      cfg,
	}
}

func (s *InflowService) FetchInflowsPaginated(c *gin.Context, paginationParams utils.PaginationParams) ([]models.Inflow, int, error) {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, 0, err
	}

	totalRecords, err := s.InflowRepo.CountInflows(user.ID)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	inflows, err := s.InflowRepo.FindInflows(user.ID, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	return inflows, int(totalRecords), nil
}

func (s *InflowService) FetchAllInflowsGroupedByMonth(c *gin.Context) ([]models.InflowSummary, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.InflowRepo.FindAllInflowsGroupedByMonth(user.ID)
}

func (s *InflowService) FetchAllInflowCategories(c *gin.Context) ([]models.InflowCategory, error) {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.InflowRepo.GetAllInflowCategories(user.ID)
}

func (s *InflowService) CreateInflow(c *gin.Context, inflow *models.Inflow) error {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	err = s.InflowRepo.InsertInflow(user.ID, inflow)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) CreateInflowCategory(c *gin.Context, inflowCategory *models.InflowCategory) error {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	err = s.InflowRepo.InsertInflowCategory(user.ID, inflowCategory)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) DeleteInflow(c *gin.Context, id uint) error {
	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	err = s.InflowRepo.DropInflow(user.ID, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *InflowService) DeleteInflowCategory(c *gin.Context, id uint) error {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return err
	}
	var count int64

	if err := s.InflowRepo.CountInflowsByCategory(user.ID, id, &count); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete inflow category: it is being used by %d inflow(s)", count)
	}

	err = s.InflowRepo.DropInflowCategory(user.ID, id)
	if err != nil {
		return err
	}
	return nil
}
