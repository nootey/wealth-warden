package services

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type ReoccurringActionService struct {
	ActionRepo  *repositories.ReoccurringActionsRepository
	AuthService *AuthService
}

func NewReoccurringActionService(repo *repositories.ReoccurringActionsRepository, authService *AuthService) *ReoccurringActionService {
	return &ReoccurringActionService{
		ActionRepo:  repo,
		AuthService: authService,
	}
}

func (s *ReoccurringActionService) FetchAllActionsForCategory(c *gin.Context, categoryName string) ([]models.RecurringAction, error) {

	user, err := s.AuthService.GetCurrentUser(c)
	if err != nil {
		return nil, err
	}
	return s.ActionRepo.FindAllActionsForCategory(user.ID, categoryName)
}
