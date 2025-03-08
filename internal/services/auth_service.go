package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/middleware"
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

func (s *AuthService) GetCurrentUser(c *gin.Context) (*models.User, error) {

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

		user, repoError := s.UserRepo.GetUserByID(userId, false)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}
