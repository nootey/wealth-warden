package services_test

import (
	"context"
	"fmt"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/suite"
)

type AuthServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAuthAccountServiceSuite(t *testing.T) {
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
	s.Equal("invalid credentials", err.Error())
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
	s.Equal("password and password confirmation do not match", err.Error())
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
