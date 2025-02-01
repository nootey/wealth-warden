package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/middleware"
)

type AuthService struct {
	UserRepo *repositories.UserRepository
}

func NewAuthService(
	userRepo *repositories.UserRepository,
) *AuthService {
	return &AuthService{
		UserRepo: userRepo,
	}
}

func (s *AuthService) GetCurrentUser(c *gin.Context) (*models.User, error) {

	refreshToken, err := c.Cookie("wwr")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookie: %v", err)
	}

	if refreshToken != "" {
		refreshClaims, err := middleware.DecodeFrontendToken(refreshToken, "refresh")
		if err != nil {
			return nil, fmt.Errorf("failed to decode refresh token: %v", err)
		}

		userId, decodeErr := middleware.DecodeEncryptedFrontendUserID(refreshClaims.UserID)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode user ID: %v", decodeErr)
		}

		user, repoError := s.UserRepo.GetUserByID(userId)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}
