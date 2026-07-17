package services_test

import (
	"context"
	"fmt"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/sessions"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/suite"
)

type SettingsServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestSettingsServiceSuite(t *testing.T) {
	suite.Run(t, new(SettingsServiceTestSuite))
}

func (s *SettingsServiceTestSuite) loginSuperAdmin() *models.User {
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
	}

	user, err := s.TC.App.AuthService.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		cfg.Seed.SuperAdminPassword,
		"test-agent",
		"127.0.0.1",
	)
	s.Require().NoError(err)
	s.Require().NotNil(user)
	return user
}

// Changing a user's own password must kill every other live session, or a
// stolen cookie survives the very action meant to revoke it.
func (s *SettingsServiceTestSuite) TestUpdateProfileSettings_PasswordChangeRevokesOtherSessions() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	otherDevice, err := store.Create(s.Ctx, user.ID, false, "other-device", "10.0.0.5")
	s.Require().NoError(err)

	newPassword := "NewSecurePass123!"
	err = s.TC.App.SettingsService.UpdateProfileSettings(s.Ctx, user.ID, models.ProfileSettingsReq{
		DisplayName:          user.DisplayName,
		Email:                user.Email,
		EmailUpdated:         false,
		Password:             &newPassword,
		PasswordConfirmation: &newPassword,
	})
	s.Require().NoError(err)

	_, err = store.Validate(s.Ctx, otherDevice)
	s.ErrorIs(err, sessions.ErrNotFound, "other session should have been revoked by the password change")

	// restore the seeded password so other tests/suites relying on it keep working
	restored := s.TC.App.Config.Seed.SuperAdminPassword
	err = s.TC.App.SettingsService.UpdateProfileSettings(s.Ctx, user.ID, models.ProfileSettingsReq{
		DisplayName:          user.DisplayName,
		Email:                user.Email,
		EmailUpdated:         false,
		Password:             &restored,
		PasswordConfirmation: &restored,
	})
	s.Require().NoError(err)
}

// A display-name-only update must not disturb other active sessions.
func (s *SettingsServiceTestSuite) TestUpdateProfileSettings_NoPasswordChangeKeepsOtherSessions() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	otherDevice, err := store.Create(s.Ctx, user.ID, false, "other-device", "10.0.0.5")
	s.Require().NoError(err)

	err = s.TC.App.SettingsService.UpdateProfileSettings(s.Ctx, user.ID, models.ProfileSettingsReq{
		DisplayName: user.DisplayName,
		Email:       user.Email,
	})
	s.Require().NoError(err)

	_, err = store.Validate(s.Ctx, otherDevice)
	s.NoError(err, "other session should survive a non-password profile update")
}
