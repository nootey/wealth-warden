package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/mailer"
	"wealth-warden/pkg/utils"
)

type AuthService struct {
	Config              *config.Config
	logger              *zap.Logger
	UserRepo            *repositories.UserRepository
	RoleRepo            *repositories.RolePermissionRepository
	SettingsRepo        *repositories.SettingsRepository
	loggingService      *LoggingService
	WebClientMiddleware *middleware.WebClientMiddleware
	jobDispatcher       jobs.JobDispatcher
	mailer              *mailer.Mailer
}

func NewAuthService(
	cfg *config.Config,
	logger *zap.Logger,
	userRepo *repositories.UserRepository,
	roleRepo *repositories.RolePermissionRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingService *LoggingService,
	webClientMiddleware *middleware.WebClientMiddleware,
	jobDispatcher jobs.JobDispatcher,
	mailer *mailer.Mailer,
) *AuthService {
	return &AuthService{
		Config:              cfg,
		logger:              logger,
		UserRepo:            userRepo,
		RoleRepo:            roleRepo,
		SettingsRepo:        settingsRepo,
		loggingService:      loggingService,
		WebClientMiddleware: webClientMiddleware,
		jobDispatcher:       jobDispatcher,
		mailer:              mailer,
	}
}

func (s *AuthService) log(event, email, userAgent, ip, status string, description *string, userID *int64) error {

	changes := utils.InitChanges()
	service := utils.DetermineServiceSource(userAgent)

	utils.CompareChanges("", status, changes, status)
	utils.CompareChanges("", service, changes, "service")
	utils.CompareChanges("", email, changes, "email")
	utils.CompareChanges("", utils.SafeString(&ip), changes, "ip_address")
	utils.CompareChanges("", utils.SafeString(&userAgent), changes, "user_agent")

	err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingService.Repo,
		Logger:      s.logger,
		Event:       event,
		Category:    "auth",
		Description: description,
		Payload:     changes,
		Causer:      userID,
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
		logErr := s.log("login", email, userAgent, ip, "fail", &desc, nil)
		if logErr != nil {
			return "", "", 0, logErr
		}

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(password))
	if err != nil {
		desc := "incorrect_password"
		logErr := s.log("login", email, userAgent, ip, "fail", &desc, nil)
		if logErr != nil {
			return "", "", 0, logErr
		}

		err := errors.New("invalid credentials")
		return "", "", 0, err
	}

	user, _ := s.UserRepo.FindUserByEmail(nil, email)
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

	logErr := s.log("login", email, userAgent, ip, "success", nil, &user.ID)
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

		user, repoError := s.UserRepo.FindUserByID(nil, userId)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}

func (s *AuthService) ValidateInvitation(hash string) error {
	if hash == "" {
		err := errors.New("validation token is required")
		return err
	}

	// Do additional validation if needed, for now, confirming it exists is enough
	_, err := s.UserRepo.FindUserInvitationByHash(nil, hash)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) dispatchConfirmationEmail(user *models.User) error {

	tx := s.UserRepo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Always ensure only one active token for this user/type
	if err := s.UserRepo.DeleteTokenByData(tx, "confirm_email", "user_id", user.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Insert new token
	newToken, err := s.UserRepo.InsertToken(tx, "confirm_email", "user_id", user.ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit before sending email
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Send email after commit
	if err := s.mailer.SendConfirmationEmail(user.Email, user.DisplayName, newToken.TokenValue); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) SignUp(form models.RegisterForm, userAgent, ip string) error {

	// Validation
	settings, err := s.SettingsRepo.FetchGeneralSettings(nil)
	if err != nil {
		return err
	}

	if !settings.AllowSignups {
		return errors.New("open sign ups are not currently enabled")
	}

	if form.Password != form.PasswordConfirmation {
		return errors.New("password and password confirmation do not match")
	}

	existingUser, _ := s.UserRepo.FindUserByEmail(nil, form.Email)

	password, passwordErr := utils.ValidatePasswordStrength(form.Password)
	if passwordErr != nil {
		return passwordErr
	}

	hashedPass, err := utils.HashAndSaltPassword(password)
	if err != nil {
		return err
	}

	if existingUser == nil {

		tx := s.UserRepo.DB.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		role, err := s.RoleRepo.FindRoleByName("member")
		if err != nil {
			tx.Rollback()
			return err
		}

		username := strings.ReplaceAll(strings.ToLower(form.DisplayName), " ", "")

		user := &models.User{
			DisplayName: form.DisplayName,
			Username:    username,
			Email:       form.Email,
			Password:    hashedPass,
			RoleID:      role.ID,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		_, err = s.UserRepo.InsertUser(tx, user)
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}

		desc := "Via open signup"
		logErr := s.log("register", user.Email, userAgent, ip, "success", &desc, &user.ID)
		if logErr != nil {
			return logErr
		}

		err = s.dispatchConfirmationEmail(user)
		if err != nil {
			return err
		}

	}

	return nil
}

func (s *AuthService) ResendConfirmationEmail(email, userAgent, ip string) error {

	user, err := s.UserRepo.FindUserByEmail(nil, email)
	if err != nil {
		return err
	}

	if user.ID == 0 {
		return errors.New("no user found for given email")
	}

	err = s.dispatchConfirmationEmail(user)
	if err != nil {
		return err
	}

	desc := "Requested a resend"
	logErr := s.log("confirm-email", user.Email, userAgent, ip, "success", &desc, &user.ID)
	if logErr != nil {
		return logErr
	}

	return nil
}

func (s *AuthService) ConfirmEmail(tokenValue, userAgent, ip string) error {

	tx := s.UserRepo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	token, err := s.UserRepo.FindTokenByValue(tx, "confirm_email", tokenValue)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if token == nil {
		_ = tx.Rollback()
		return errors.New("no valid token found")
	}

	raw, err := utils.UnwrapToken(token, "user_id")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("no user_id in token data")
	}

	num := raw.(json.Number)
	userID, err := num.Int64()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("invalid user_id in token data: %v", err)
	}

	user, err := s.UserRepo.FindUserByID(tx, userID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	now := time.Now().UTC()
	user.EmailConfirmed = &now

	if err := tx.Save(&user).Error; err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := s.UserRepo.DeleteTokenByData(tx, "confirm_email", "user_id", userID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	logErr := s.log("confirm-email", user.Email, userAgent, ip, "success", nil, &user.ID)
	if logErr != nil {
		return logErr
	}

	return nil
}

func (s *AuthService) RegisterUser(form models.RegisterForm, userAgent, ip string) error {

	return nil
}
