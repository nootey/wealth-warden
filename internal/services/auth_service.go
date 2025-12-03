package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/mailer"
	"wealth-warden/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

type AuthServiceInterface interface {
	LoginUser(ctx context.Context, email, password, userAgent, ip string, rememberMe bool) (string, string, int, error)
	GetCurrentUser(ctx context.Context, refreshToken string) (*models.User, error)
	ValidateInvitation(ctx context.Context, hash string) error
	SignUp(ctx context.Context, form models.RegisterForm, userAgent, ip string) error
	ResendConfirmationEmail(ctx context.Context, email, userAgent, ip string) error
	ConfirmEmail(ctx context.Context, tokenValue, userAgent, ip string) error
	RequestPasswordReset(ctx context.Context, email, userAgent, ip string) error
	ValidatePasswordReset(ctx context.Context, tokenValue string) (string, error)
	ResetPassword(ctx context.Context, form models.ResetPasswordForm, userAgent, ip string) error
}
type AuthService struct {
	userRepo            repositories.UserRepositoryInterface
	roleRepo            repositories.RolePermissionRepositoryInterface
	settingsRepo        repositories.SettingsRepositoryInterface
	loggingRepo         repositories.LoggingRepositoryInterface
	webClientMiddleware *middleware.WebClientMiddleware
	jobDispatcher       jobs.JobDispatcher
	mailer              *mailer.Mailer
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	roleRepo *repositories.RolePermissionRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	webClientMiddleware *middleware.WebClientMiddleware,
	jobDispatcher jobs.JobDispatcher,
	mailer *mailer.Mailer,
) *AuthService {
	return &AuthService{
		userRepo:            userRepo,
		roleRepo:            roleRepo,
		settingsRepo:        settingsRepo,
		loggingRepo:         loggingRepo,
		webClientMiddleware: webClientMiddleware,
		jobDispatcher:       jobDispatcher,
		mailer:              mailer,
	}
}

var _ AuthServiceInterface = (*AuthService)(nil)

func (s *AuthService) log(event, email, userAgent, ip, status string, description *string, userID *int64) error {

	changes := utils.InitChanges()
	service := utils.DetermineServiceSource(userAgent)

	utils.CompareChanges("", status, changes, status)
	utils.CompareChanges("", service, changes, "service")
	utils.CompareChanges("", email, changes, "email")
	utils.CompareChanges("", utils.SafeString(&ip), changes, "ip_address")
	utils.CompareChanges("", utils.SafeString(&userAgent), changes, "user_agent")

	err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *AuthService) dispatchConfirmationEmail(ctx context.Context, user *models.User) error {

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}

	// Always ensure only one active token for this user/type
	if err := s.userRepo.DeleteTokenByData(ctx, tx, "confirm-email", "user_id", user.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Insert new token
	newToken, err := s.userRepo.InsertToken(ctx, tx, "confirm-email", "user_id", user.ID)
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

func (s *AuthService) dispatchPasswordResetEmail(ctx context.Context, user *models.User) error {

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}

	// Always ensure only one active token for this user/type
	if err := s.userRepo.DeleteTokenByData(ctx, tx, "password-reset", "user_id", user.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Insert new token
	newToken, err := s.userRepo.InsertToken(ctx, tx, "password-reset", "user_id", user.ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit before sending email
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Send email after commit
	if err := s.mailer.SendPasswordResetEmail(user.Email, user.DisplayName, newToken.TokenValue); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) LoginUser(ctx context.Context, email, password, userAgent, ip string, rememberMe bool) (string, string, int, error) {

	userPassword, _ := s.userRepo.GetPasswordByEmail(ctx, nil, email)
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

	user, _ := s.userRepo.FindUserByEmail(ctx, nil, email)
	if user == nil {
		err = errors.New("user data unavailable")
		return "", "", 0, err
	}

	accessToken, refreshToken, err := s.webClientMiddleware.GenerateLoginTokens(user.ID, rememberMe)
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

func (s *AuthService) GetCurrentUser(ctx context.Context, refreshToken string) (*models.User, error) {

	if refreshToken != "" {
		refreshClaims, err := s.webClientMiddleware.DecodeWebClientToken(refreshToken, "refresh")
		if err != nil {
			return nil, fmt.Errorf("failed to decode refresh token: %v", err)
		}

		userId, decodeErr := s.webClientMiddleware.DecodeWebClientUserID(refreshClaims.UserID)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode user ID: %v", decodeErr)
		}

		user, repoError := s.userRepo.FindUserByID(ctx, nil, userId)
		if repoError != nil {
			return nil, fmt.Errorf("failed to get user from repository: %v", repoError)
		}

		return user, nil
	}

	return nil, fmt.Errorf("no refresh token found")
}

func (s *AuthService) ValidateInvitation(ctx context.Context, hash string) error {
	if hash == "" {
		err := errors.New("validation token is required")
		return err
	}

	// Do additional validation if needed, for now, confirming it exists is enough
	_, err := s.userRepo.FindUserInvitationByHash(ctx, nil, hash)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) SignUp(ctx context.Context, form models.RegisterForm, userAgent, ip string) error {

	// Validation
	settings, err := s.settingsRepo.FetchGeneralSettings(ctx, nil)
	if err != nil {
		return err
	}

	if !settings.AllowSignups {
		return errors.New("open sign ups are not currently enabled")
	}

	if form.Password != form.PasswordConfirmation {
		return errors.New("password and password confirmation do not match")
	}

	existingUser, _ := s.userRepo.FindUserByEmail(ctx, nil, form.Email)

	password, passwordErr := utils.ValidatePasswordStrength(form.Password)
	if passwordErr != nil {
		return passwordErr
	}

	hashedPass, err := utils.HashAndSaltPassword(password)
	if err != nil {
		return err
	}

	if existingUser == nil {

		tx, err := s.userRepo.BeginTx(ctx)
		if err != nil {
			return err
		}

		role, err := s.roleRepo.FindRoleByName(ctx, tx, "member")
		if err != nil {
			tx.Rollback()
			return err
		}

		user := &models.User{
			DisplayName: form.DisplayName,
			Email:       form.Email,
			Password:    hashedPass,
			RoleID:      role.ID,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		_, err = s.userRepo.InsertUser(ctx, tx, user)
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

		err = s.dispatchConfirmationEmail(ctx, user)
		if err != nil {
			return err
		}

	}

	return nil
}

func (s *AuthService) ResendConfirmationEmail(ctx context.Context, email, userAgent, ip string) error {

	user, err := s.userRepo.FindUserByEmail(ctx, nil, email)
	if err != nil {
		return err
	}

	if user.ID == 0 {
		return errors.New("no user found for given email")
	}

	err = s.dispatchConfirmationEmail(ctx, user)
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

func (s *AuthService) ConfirmEmail(ctx context.Context, tokenValue, userAgent, ip string) error {

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}

	token, err := s.userRepo.FindTokenByValue(ctx, tx, "confirm-email", tokenValue)
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

	user, err := s.userRepo.FindUserByID(ctx, tx, userID)
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

	if err := s.userRepo.DeleteTokenByData(ctx, tx, "confirm-email", "user_id", userID); err != nil {
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

func (s *AuthService) RequestPasswordReset(ctx context.Context, email, userAgent, ip string) error {

	user, err := s.userRepo.FindUserByEmail(ctx, nil, email)
	if err != nil {
		return err
	}

	if user.ID == 0 {
		return errors.New("no user found for given email")
	}

	err = s.dispatchPasswordResetEmail(ctx, user)
	if err != nil {
		return err
	}

	desc := "Requested a password reset"
	logErr := s.log("password-reset", user.Email, userAgent, ip, "success", &desc, &user.ID)
	if logErr != nil {
		return logErr
	}

	return nil
}

func (s *AuthService) ValidatePasswordReset(ctx context.Context, tokenValue string) (string, error) {

	token, err := s.userRepo.FindTokenByValue(ctx, nil, "password-reset", tokenValue)
	if err != nil {
		return "", err
	}

	if token == nil {
		return "", errors.New("no valid token found")
	}

	raw, err := utils.UnwrapToken(token, "user_id")
	if err != nil {
		return "", fmt.Errorf("no user_id in token data")
	}

	num := raw.(json.Number)
	userID, err := num.Int64()
	if err != nil {
		return "", fmt.Errorf("invalid user_id in token data: %v", err)
	}

	user, err := s.userRepo.FindUserByID(ctx, nil, userID)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", errors.New("user not found for given token")
	}

	return token.TokenValue, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, form models.ResetPasswordForm, userAgent, ip string) error {

	if form.Password != form.PasswordConfirmation {
		return errors.New("password and password confirmation do not match")
	}

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByEmail(ctx, tx, form.Email)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	password, passwordErr := utils.ValidatePasswordStrength(form.Password)
	if passwordErr != nil {
		return passwordErr
	}

	hashedPass, err := utils.HashAndSaltPassword(password)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = s.userRepo.UpdateUserPassword(ctx, tx, user.ID, hashedPass)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := s.userRepo.DeleteTokenByData(ctx, tx, "password-reset", "user_id", user.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	desc := "Reset password successfully"
	logErr := s.log("password-reset", user.Email, userAgent, ip, "success", &desc, &user.ID)
	if logErr != nil {
		return logErr
	}

	return nil
}

func (s *AuthService) RegisterUser(ctx context.Context, form models.RegisterForm, userAgent, ip string) error {

	return nil
}
