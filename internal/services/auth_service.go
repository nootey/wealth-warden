package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/utils"
)

type AuthService struct {
	Config              *config.Config
	logger              *zap.Logger
	UserRepo            *repositories.UserRepository
	loggingService      *LoggingService
	WebClientMiddleware *middleware.WebClientMiddleware
	jobDispatcher       jobs.JobDispatcher
}

func NewAuthService(
	cfg *config.Config,
	logger *zap.Logger,
	userRepo *repositories.UserRepository,
	loggingService *LoggingService,
	webClientMiddleware *middleware.WebClientMiddleware,
	jobDispatcher jobs.JobDispatcher,
) *AuthService {
	return &AuthService{
		Config:              cfg,
		logger:              logger,
		UserRepo:            userRepo,
		loggingService:      loggingService,
		WebClientMiddleware: webClientMiddleware,
		jobDispatcher:       jobDispatcher,
	}
}

func (s *AuthService) logLoginAttempt(email, userAgent, ip, status string, description *string, user *models.User) error {

	changes := utils.InitChanges()
	service := utils.DetermineServiceSource(userAgent)

	utils.CompareChanges("", status, changes, status)
	utils.CompareChanges("", service, changes, "service")
	utils.CompareChanges("", email, changes, "email")
	utils.CompareChanges("", utils.SafeString(&ip), changes, "ip_address")
	utils.CompareChanges("", utils.SafeString(&userAgent), changes, "user_agent")

	err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingService.LoggingRepo,
		Logger:      s.logger,
		Event:       "login",
		Category:    "auth",
		Description: description,
		Payload:     changes,
		Causer:      user,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) LoginUser(email, password, userAgent, ip string, rememberMe bool) (string, string, int, error) {

	userPassword, _ := s.UserRepo.GetPasswordByEmail(email)
	if userPassword == "" {
		desc := "user does not exist"
		logErr := s.logLoginAttempt(email, userAgent, ip, "fail", &desc, nil)
		if logErr != nil {
			return "", "", 0, logErr
		}

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(password))
	if err != nil {
		desc := "incorrect_password"
		logErr := s.logLoginAttempt(email, userAgent, ip, "fail", &desc, nil)
		if logErr != nil {
			return "", "", 0, logErr
		}

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	user, _ := s.UserRepo.GetUserByEmail(email)
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
		expiresAt = int(constants.RefreshCookieTTLLong.Seconds())
	} else {
		expiresAt = int(constants.RefreshCookieTTLShort.Seconds())
	}

	logErr := s.logLoginAttempt(email, userAgent, ip, "success", nil, user)
	if logErr != nil {
		return "", "", 0, logErr
	}

	return accessToken, refreshToken, expiresAt, nil
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

		user, repoError := s.UserRepo.GetUserByID(userId)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}
