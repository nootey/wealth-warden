package services

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type TransactionService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.TransactionRepository
}

func NewTransactionService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.TransactionRepository,
) *TransactionService {
	return &TransactionService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}
func (s *TransactionService) FetchAllCategories(c *gin.Context) ([]models.Category, error) {
	return s.Repo.FindAllCategories(nil)
}
