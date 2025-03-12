package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type AuthService struct {
	Config              *config.Config
	UserRepo            *repositories.UserRepository
	LoggingService      *LoggingService
	WebClientMiddleware *middleware.WebClientMiddleware
}

func NewAuthService(
	cfg *config.Config,
	userRepo *repositories.UserRepository,
	loggingService *LoggingService,
	webClientMiddleware *middleware.WebClientMiddleware,
) *AuthService {
	return &AuthService{
		Config:              cfg,
		UserRepo:            userRepo,
		LoggingService:      loggingService,
		WebClientMiddleware: webClientMiddleware,
	}
}

func (s *AuthService) GetCurrentUser(c *gin.Context, withSecrets bool) (*models.User, error) {

	refreshToken, err := c.Cookie("refresh")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookie: %v", err)
	}

	if refreshToken != "" {
		refreshClaims, err := s.WebClientMiddleware.DecodeWebClientToken(refreshToken, "refresh")
		if err != nil {
			return nil, fmt.Errorf("failed to decode refresh token: %v", err)
		}

		userId, decodeErr := s.WebClientMiddleware.DecodeWebClientUserID(refreshClaims.UserID)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode user ID: %v", decodeErr)
		}

		user, repoError := s.UserRepo.GetUserByID(userId, withSecrets)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}

func (s *AuthService) UpdateBudgetInitializedStatus(tx *gorm.DB, user *models.User, budgetStatus bool) error {

	newTx := false
	if tx == nil {
		tx = s.UserRepo.DB.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		newTx = true
	}

	err := s.UserRepo.UpdateUserSecret(tx, user, "budget_initialized", budgetStatus)
	if err != nil {
		return err
	}

	if newTx {
		return tx.Commit().Error
	}
	return nil
}
