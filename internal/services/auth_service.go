package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type AuthService struct {
	Config              *config.Config
	logger              *zap.Logger
	UserRepo            *repositories.UserRepository
	loggingService      *LoggingService
	WebClientMiddleware *middleware.WebClientMiddleware
}

func NewAuthService(
	cfg *config.Config,
	logger *zap.Logger,
	userRepo *repositories.UserRepository,
	loggingService *LoggingService,
	webClientMiddleware *middleware.WebClientMiddleware,
) *AuthService {
	return &AuthService{
		Config:              cfg,
		logger:              logger,
		UserRepo:            userRepo,
		loggingService:      loggingService,
		WebClientMiddleware: webClientMiddleware,
	}
}

func (s *AuthService) logLoginAttempt(email, userAgent, ip, status string, description *string, user *models.User) {
	go func(email, userAgent, ip, status string, description *string, user *models.User) {
		changes := utils.InitChanges()
		service := utils.DetermineServiceSource(userAgent)

		utils.CompareChanges("", service, changes, "service")
		utils.CompareChanges("", email, changes, "email")
		utils.CompareChanges("", utils.SafeString(&ip), changes, "ip_address")
		utils.CompareChanges("", utils.SafeString(&userAgent), changes, "user_agent")

		if err := s.loggingService.LoggingRepo.InsertAccessLog(nil, status, "login", user, changes, description); err != nil {
			s.logger.Error("failed to insert access log", zap.Error(err))
		}
	}(email, userAgent, ip, status, description, user)
}

func (s *AuthService) LoginUser(email, password, userAgent, ip string, rememberMe bool) (string, string, int, error) {

	userPassword, _ := s.UserRepo.GetPasswordByEmail(email)
	if userPassword == "" {
		desc := "user does not exist"
		s.logLoginAttempt(email, userAgent, ip, "fail", &desc, nil)

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(password))
	if err != nil {
		desc := "incorrect_password"
		s.logLoginAttempt(email, userAgent, ip, "fail", &desc, nil)

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	user, _ := s.UserRepo.GetUserByEmail(email, false)
	if user == nil {
		err = errors.New("user data unavailable")
		return "", "", 0, err
	}

	accessToken, refreshToken, err := s.WebClientMiddleware.GenerateLoginTokens(user.ID, rememberMe)
	if err != nil {
		return "", "", 0, err
	}

	var expiresAt int
	if rememberMe {
		expiresAt = 3600 * 24 * 14
	} else {
		expiresAt = 3600 * 24
	}

	s.logLoginAttempt(email, userAgent, ip, "success", nil, user)

	return accessToken, refreshToken, expiresAt, nil
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
