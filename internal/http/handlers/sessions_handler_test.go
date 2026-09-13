package handlers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/services"
	"wealth-warden/internal/sessions"
	"wealth-warden/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type SessionsHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *mocks.MockSessionsServiceInterface
	handler     *handlers.SessionsHandler
}

func (suite *SessionsHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockSessionsServiceInterface(suite.T())
	suite.handler = handlers.NewSessionsHandler(suite.mockService)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.DELETE("/sessions/:id", suite.handler.RevokeSession)
}

func (suite *SessionsHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SessionsHandlerTestSuite) revoke(returned error) *httptest.ResponseRecorder {
	suite.mockService.On("RevokeSession", mock.Anything, int64(123), "", "abc").
		Return(returned)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/sessions/abc", nil)
	suite.router.ServeHTTP(w, req)
	return w
}

// the handler no longer branches on the error, so the sentinel carries the status
func (suite *SessionsHandlerTestSuite) TestRevokeSession_CurrentSession() {
	w := suite.revoke(services.ErrCannotRevokeCurrentSession)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("log out to end the current session", response["message"])
}

func (suite *SessionsHandlerTestSuite) TestRevokeSession_NotFound() {
	w := suite.revoke(sessions.ErrNotFound)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("session not found", response["message"])
}

// a sentinel still resolves after the service wraps it with context
func (suite *SessionsHandlerTestSuite) TestRevokeSession_WrappedSentinel() {
	w := suite.revoke(fmt.Errorf("revoke failed: %w", sessions.ErrNotFound))

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *SessionsHandlerTestSuite) TestRevokeSession_UnclassifiedDoesNotLeak() {
	w := suite.revoke(errors.New("redis: connection refused"))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "redis")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

func TestSessionsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SessionsHandlerTestSuite))
}
