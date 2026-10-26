//go:build integration

package services_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"wealth-warden/internal/config"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/authz"

	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

type AuthServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationAuthAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}

func (s *AuthServiceTestSuite) TestValidateLogin_Success() {

	svc := s.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
	}

	user, err := svc.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		cfg.Seed.SuperAdminPassword,
		"test-agent",
		"127.0.0.1",
	)

	s.NoError(err)
	s.NotNil(user)
	s.Equal(cfg.Seed.SuperAdminEmail, user.Email)
}

func (s *AuthServiceTestSuite) TestValidateLogin_WrongPassword() {
	svc := s.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
	}

	user, err := svc.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		"wrongpassword123",
		"test-agent",
		"127.0.0.1",
	)

	s.Error(err)
	s.Nil(user)
	s.True(errors.Is(err, services.ErrInvalidCredentials))
}

func (s *AuthServiceTestSuite) TestPermsForUser_CachesRoleUntilTTL() {
	ctx := context.Background()
	cfg, err := config.LoadConfig(nil)
	s.Require().NoError(err)

	var user models.User
	s.Require().NoError(s.TC.DB.Where("email = ?", cfg.Seed.SuperAdminEmail).First(&user).Error)
	var memberRoleID int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&memberRoleID).Error)

	cached := authz.NewService(s.TC.DB, time.Hour)
	expired := authz.NewService(s.TC.DB, 0)
	_, err = cached.PermsForUser(ctx, user.ID)
	s.Require().NoError(err)

	s.Require().NoError(s.TC.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("role_id", memberRoleID).Error)
	defer s.TC.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("role_id", user.RoleID)

	perms, err := cached.PermsForUser(ctx, user.ID)
	s.NoError(err)
	s.Contains(perms, "root_access")

	perms, err = expired.PermsForUser(ctx, user.ID)
	s.NoError(err)
	s.NotContains(perms, "root_access")
}

func (s *AuthServiceTestSuite) setUserRole(user models.User, roleID int64) {
	_, err := s.TC.App.UserService.UpdateUser(s.Ctx, user.ID, user.ID, &models.UserReq{
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		RoleID:         roleID,
		EmailConfirmed: user.EmailConfirmed,
	})
	s.Require().NoError(err)
}

func (s *AuthServiceTestSuite) TestUpdateUser_RoleChangeClearsCachedRole() {
	authzSvc := s.TC.App.AuthzService
	cfg, err := config.LoadConfig(nil)
	s.Require().NoError(err)

	var user models.User
	s.Require().NoError(s.TC.DB.Where("email = ?", cfg.Seed.SuperAdminEmail).First(&user).Error)
	var memberRoleID int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&memberRoleID).Error)

	perms, err := authzSvc.PermsForUser(s.Ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Contains(perms, "root_access")

	s.setUserRole(user, memberRoleID)
	defer s.setUserRole(user, user.RoleID)

	perms, err = authzSvc.PermsForUser(s.Ctx, user.ID)
	s.NoError(err)
	s.NotContains(perms, "root_access")
}

func (s *AuthServiceTestSuite) TestUpdateUser_ValidatesPassword() {
	var memberRoleID int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&memberRoleID).Error)
	user := models.User{Email: "password-change@example.com", Password: "x", DisplayName: "password change", RoleID: memberRoleID}
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Create(&user).Error)

	updatePassword := func(password string, confirmation *string) error {
		_, err := s.TC.App.UserService.UpdateUser(s.Ctx, user.ID, user.ID, &models.UserReq{
			Email:                user.Email,
			DisplayName:          user.DisplayName,
			RoleID:               user.RoleID,
			Password:             &password,
			PasswordConfirmation: confirmation,
		})
		return err
	}
	login := func(password string) error {
		_, err := s.TC.App.AuthService.ValidateLogin(s.Ctx, user.Email, password, "test-agent", "127.0.0.1")
		return err
	}

	strong, mismatch, weak := "SecurePassword123!", "DifferentPassword123!", "weak"

	s.Error(updatePassword(strong, nil))
	s.Error(updatePassword(strong, &mismatch))
	s.Error(updatePassword(weak, &weak))
	s.Error(login(strong))

	s.NoError(updatePassword("", nil), "an empty password must skip the change")

	s.Require().NoError(updatePassword(strong, &strong))
	s.NoError(login(strong))
}

func (s *AuthServiceTestSuite) TestUpdateRole_PermissionChangeClearsCachedPerms() {
	authzSvc := s.TC.App.AuthzService
	roleSvc := s.TC.App.RoleService
	cfg, err := config.LoadConfig(nil)
	s.Require().NoError(err)

	var user models.User
	s.Require().NoError(s.TC.DB.Where("email = ?", cfg.Seed.SuperAdminEmail).First(&user).Error)
	var viewData, manageData models.Permission
	s.Require().NoError(s.TC.DB.Where("name = ?", "view_data").First(&viewData).Error)
	s.Require().NoError(s.TC.DB.Where("name = ?", "manage_data").First(&manageData).Error)

	roleID, err := roleSvc.InsertRole(s.Ctx, user.ID, models.RoleReq{Name: "Cache Probe", Permissions: []models.Permission{viewData}})
	s.Require().NoError(err)
	defer func() { s.NoError(roleSvc.DeleteRole(s.Ctx, user.ID, roleID)) }()

	s.setUserRole(user, roleID)
	defer s.setUserRole(user, user.RoleID)

	perms, err := authzSvc.PermsForUser(s.Ctx, user.ID)
	s.Require().NoError(err)
	s.Require().NotContains(perms, "manage_data")

	_, err = roleSvc.UpdateRole(s.Ctx, user.ID, roleID, &models.RoleReq{Name: "Cache Probe", Permissions: []models.Permission{viewData, manageData}})
	s.Require().NoError(err)

	perms, err = authzSvc.PermsForUser(s.Ctx, user.ID)
	s.NoError(err)
	s.Contains(perms, "manage_data")
}

func (s *AuthServiceTestSuite) TestGetCurrentUser() {
	svc := s.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
	}

	user, err := svc.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		cfg.Seed.SuperAdminPassword,
		"test-agent",
		"127.0.0.1",
	)

	s.NoError(err)
	s.NotNil(user)

	// Fetch the same user by ID
	currentUser, err := svc.GetCurrentUser(context.Background(), user.ID)

	s.NoError(err)
	s.NotNil(currentUser)
	s.Equal(user.ID, currentUser.ID)
	s.Equal(user.Email, currentUser.Email)
}

func (s *AuthServiceTestSuite) TestGetCurrentUser_InvalidUserID() {
	svc := s.TC.App.AuthService

	// Try to fetch user with non-existent ID
	currentUser, err := svc.GetCurrentUser(context.Background(), 999999)

	s.Error(err)
	s.Nil(currentUser)
}

func (s *AuthServiceTestSuite) TestSignUp_Success() {
	svc := s.TC.App.AuthService

	form := models.RegisterForm{
		AuthForm: models.AuthForm{
			Email:    "newuser@example.com",
			Password: "SecurePassword123!",
		},
		DisplayName:          "New User",
		PasswordConfirmation: "SecurePassword123!",
	}

	userID, err := svc.SignUp(
		context.Background(),
		form,
		"test-agent",
		"127.0.0.1",
	)

	s.NoError(err)
	s.NotZero(userID)

	// Verify user was created
	user, err := svc.GetCurrentUser(context.Background(), userID)
	s.NoError(err)
	s.Equal(form.Email, user.Email)
	s.Equal(form.DisplayName, user.DisplayName)
	// Dev defaults have no mailer, so the user is auto-confirmed on signup.
	s.NotNil(user.EmailConfirmed)
}

func (s *AuthServiceTestSuite) TestSignUp_PasswordMismatch() {
	svc := s.TC.App.AuthService

	form := models.RegisterForm{
		AuthForm: models.AuthForm{
			Email:    "newuser2@example.com",
			Password: "SecurePassword123!",
		},
		DisplayName:          "New User",
		PasswordConfirmation: "DifferentPassword123!",
	}

	userID, err := svc.SignUp(
		context.Background(),
		form,
		"test-agent",
		"127.0.0.1",
	)

	s.Error(err)
	s.Zero(userID)
	s.True(errors.Is(err, services.ErrPasswordMismatch))
}

func (s *AuthServiceTestSuite) TestSignUp_WeakPassword() {
	svc := s.TC.App.AuthService

	form := models.RegisterForm{
		AuthForm: models.AuthForm{
			Email:    "newuser3@example.com",
			Password: "weak",
		},
		DisplayName:          "New User",
		PasswordConfirmation: "weak",
	}

	userID, err := svc.SignUp(
		context.Background(),
		form,
		"test-agent",
		"127.0.0.1",
	)

	s.Error(err)
	s.Zero(userID)
}

// a reset must be tied to a token; email alone must never change a password
func (s *AuthServiceTestSuite) TestResetPassword_UnknownToken() {
	svc := s.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	s.Require().NoError(err)

	err = svc.ResetPassword(s.Ctx, models.ResetPasswordForm{
		AuthForm:             models.AuthForm{Email: cfg.Seed.SuperAdminEmail, Password: "NewSecurePassword123!"},
		PasswordConfirmation: "NewSecurePassword123!",
		Token:                "not-a-real-token",
	}, "test-agent", "127.0.0.1")

	s.True(errors.Is(err, services.ErrInvalidToken))

	// the old password still works
	_, err = svc.ValidateLogin(s.Ctx, cfg.Seed.SuperAdminEmail, cfg.Seed.SuperAdminPassword, "test-agent", "127.0.0.1")
	s.NoError(err)
}

func (s *AuthServiceTestSuite) TestResetPassword_Success() {
	svc := s.TC.App.AuthService

	userID, err := svc.SignUp(s.Ctx, models.RegisterForm{
		AuthForm:             models.AuthForm{Email: "reset@example.com", Password: "OldPassword123!"},
		DisplayName:          "Reset User",
		PasswordConfirmation: "OldPassword123!",
	}, "test-agent", "127.0.0.1")
	s.Require().NoError(err)

	token := models.Token{
		TokenType:  "password-reset",
		TokenValue: "valid-reset-token",
		Data:       datatypes.JSONMap{"user_id": userID},
	}
	s.Require().NoError(s.TC.DB.Create(&token).Error)

	err = svc.ResetPassword(s.Ctx, models.ResetPasswordForm{
		AuthForm:             models.AuthForm{Email: "reset@example.com", Password: "NewPassword123!"},
		PasswordConfirmation: "NewPassword123!",
		Token:                token.TokenValue,
	}, "test-agent", "127.0.0.1")
	s.NoError(err)

	_, err = svc.ValidateLogin(s.Ctx, "reset@example.com", "NewPassword123!", "test-agent", "127.0.0.1")
	s.NoError(err)

	// the token is single use
	_, err = svc.ValidatePasswordReset(s.Ctx, token.TokenValue)
	s.True(errors.Is(err, services.ErrInvalidToken))
}
