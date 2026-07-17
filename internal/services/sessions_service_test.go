package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/internal/sessions"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/suite"
)

type SessionsServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestSessionsServiceSuite(t *testing.T) {
	suite.Run(t, new(SessionsServiceTestSuite))
}

func (s *SessionsServiceTestSuite) loginSuperAdmin() *models.User {
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

// The current session must sort first regardless of last-seen, and the rest
// fall back to most-recently-seen order.
func (s *SessionsServiceTestSuite) TestListSessions_CurrentFirstThenByLastSeen() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	_, err := store.Create(s.Ctx, user.ID, false, "older-device", "10.0.0.1")
	s.Require().NoError(err)
	// last_seen has 1-second resolution; space out creation to get a stable order.
	time.Sleep(1100 * time.Millisecond)
	current, err := store.Create(s.Ctx, user.ID, false, "current-device", "10.0.0.2")
	s.Require().NoError(err)
	time.Sleep(1100 * time.Millisecond)
	_, err = store.Create(s.Ctx, user.ID, false, "newer-device", "10.0.0.3")
	s.Require().NoError(err)

	list, err := s.TC.App.SessionsService.ListSessions(s.Ctx, user.ID, current)
	s.Require().NoError(err)
	s.Require().Len(list, 3)

	s.True(list[0].Current, "current session must be first")
	s.False(list[1].Current)
	s.False(list[2].Current)
	s.True(list[1].LastSeen.After(list[2].LastSeen) || list[1].LastSeen.Equal(list[2].LastSeen),
		"non-current sessions must be ordered by most recently seen")
}

// A user must not be able to revoke the session they're currently authenticated with.
func (s *SessionsServiceTestSuite) TestRevokeSession_CannotRevokeCurrent() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	current, err := store.Create(s.Ctx, user.ID, false, "current-device", "10.0.0.1")
	s.Require().NoError(err)

	list, err := s.TC.App.SessionsService.ListSessions(s.Ctx, user.ID, current)
	s.Require().NoError(err)
	handle := list[0].ID

	err = s.TC.App.SessionsService.RevokeSession(s.Ctx, user.ID, current, handle)
	s.ErrorIs(err, services.ErrCannotRevokeCurrentSession)

	_, err = store.Validate(s.Ctx, current)
	s.NoError(err, "current session must survive the rejected revoke")
}

// Revoking a non-current session by its handle deletes exactly that session.
func (s *SessionsServiceTestSuite) TestRevokeSession_RemovesOtherSession() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	current, err := store.Create(s.Ctx, user.ID, false, "current-device", "10.0.0.1")
	s.Require().NoError(err)
	other, err := store.Create(s.Ctx, user.ID, false, "other-device", "10.0.0.2")
	s.Require().NoError(err)

	list, err := s.TC.App.SessionsService.ListSessions(s.Ctx, user.ID, current)
	s.Require().NoError(err)

	var otherHandle string
	for _, info := range list {
		if !info.Current {
			otherHandle = info.ID
		}
	}
	s.Require().NotEmpty(otherHandle)

	err = s.TC.App.SessionsService.RevokeSession(s.Ctx, user.ID, current, otherHandle)
	s.Require().NoError(err)

	_, err = store.Validate(s.Ctx, other)
	s.ErrorIs(err, sessions.ErrNotFound)

	_, err = store.Validate(s.Ctx, current)
	s.NoError(err, "current session must be unaffected")
}

// An unknown handle (already-revoked or forged) must surface as not-found.
func (s *SessionsServiceTestSuite) TestRevokeSession_UnknownHandleNotFound() {
	user := s.loginSuperAdmin()
	current, err := s.TC.App.SessionStore.Create(s.Ctx, user.ID, false, "current-device", "10.0.0.1")
	s.Require().NoError(err)

	err = s.TC.App.SessionsService.RevokeSession(s.Ctx, user.ID, current, "not-a-real-handle")
	s.ErrorIs(err, sessions.ErrNotFound)
}

// Revoking all sessions must remove every session for the user, including the current one.
func (s *SessionsServiceTestSuite) TestRevokeAllSessions_RemovesEverySession() {
	user := s.loginSuperAdmin()
	store := s.TC.App.SessionStore

	current, err := store.Create(s.Ctx, user.ID, false, "current-device", "10.0.0.1")
	s.Require().NoError(err)
	other, err := store.Create(s.Ctx, user.ID, false, "other-device", "10.0.0.2")
	s.Require().NoError(err)

	err = s.TC.App.SessionsService.RevokeAllSessions(s.Ctx, user.ID)
	s.Require().NoError(err)

	_, err = store.Validate(s.Ctx, current)
	s.ErrorIs(err, sessions.ErrNotFound)
	_, err = store.Validate(s.Ctx, other)
	s.ErrorIs(err, sessions.ErrNotFound)
}
