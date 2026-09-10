package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/config"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type AuthHandlerTestSuite struct {
	suite.Suite
	router         *gin.Engine
	mockService    *mocks.MockAuthServiceInterface
	mockMiddleware *mocks.MockWebClientMiddlewareInterface
	mockConfig     *config.Config
	handler        *handlers.AuthHandler
}

func (suite *AuthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockAuthServiceInterface(suite.T())
	suite.mockMiddleware = mocks.NewMockWebClientMiddlewareInterface(suite.T())

	suite.handler = handlers.NewAuthHandler(
		suite.mockConfig,
		suite.mockMiddleware,
		suite.mockService,
	)

	suite.router = gin.New()

	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.POST("/auth/login", suite.handler.LoginUser)
	suite.router.POST("/auth/signup", suite.handler.SignUp)
	suite.router.GET("/auth/user", suite.handler.GetAuthUser)
	suite.router.POST("/auth/logout", suite.handler.LogoutUser)
	suite.router.POST("/auth/reset-password", suite.handler.ResetPassword)
}

func (suite *AuthHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockMiddleware.AssertExpectations(suite.T())
}

func TestAuthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}

func (suite *AuthHandlerTestSuite) TestLoginUser_Success() {

	form := models.LoginForm{
		AuthForm: models.AuthForm{
			Email:    "test@example.com",
			Password: "password123",
		},
		RememberMe: false,
	}

	user := &models.User{
		ID:    123,
		Email: "test@example.com",
	}

	suite.mockService.On("ValidateLogin",
		mock.Anything,
		form.Email,
		form.Password,
		mock.Anything,
		mock.Anything,
	).Return(user, nil)

	suite.mockMiddleware.On("CreateLoginSession", mock.Anything, user.ID, form.RememberMe, mock.Anything, mock.Anything).
		Return("session-id", 86400, nil)
	suite.mockMiddleware.On("CookieDomainForEnv").Return("localhost")
	suite.mockMiddleware.On("CookieSecure").Return(false)

	body, _ := json.Marshal(form)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *AuthHandlerTestSuite) TestLoginUser_InvalidJSON() {

	invalidJSON := []byte(`{"email": "test@example.com", "password":}`)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AuthHandlerTestSuite) TestLoginUser_SessionCreationFails() {
	form := models.LoginForm{
		AuthForm: models.AuthForm{
			Email:    "test@example.com",
			Password: "password123",
		},
		RememberMe: false,
	}

	user := &models.User{
		ID:    123,
		Email: "test@example.com",
	}

	suite.mockService.On("ValidateLogin",
		mock.Anything,
		form.Email,
		form.Password,
		mock.Anything,
		mock.Anything,
	).Return(user, nil)

	suite.mockMiddleware.On("CreateLoginSession", mock.Anything, user.ID, form.RememberMe, mock.Anything, mock.Anything).
		Return("", 0, errors.New("session creation failed"))

	body, _ := json.Marshal(form)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *AuthHandlerTestSuite) TestSignUp_Success() {

	form := models.RegisterForm{
		AuthForm: models.AuthForm{
			Email:    "test@example.com",
			Password: "password123",
		},
		DisplayName:          "Test User",
		PasswordConfirmation: "password123",
	}

	suite.mockService.On("SignUp",
		mock.Anything,
		form,
		mock.Anything,
		mock.Anything,
	).Return(int64(1), nil)

	body, _ := json.Marshal(form)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *AuthHandlerTestSuite) TestGetAuthUser_Success() {

	user := &models.User{
		ID:    123,
		Email: "test@example.com",
	}

	suite.mockService.On("GetCurrentUser", mock.Anything, int64(123)).
		Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/user", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *AuthHandlerTestSuite) TestGetAuthUser_ServiceError() {

	suite.mockService.On("GetCurrentUser", mock.Anything, int64(123)).
		Return(nil, errors.New("user not found"))

	req := httptest.NewRequest(http.MethodGet, "/auth/user", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *AuthHandlerTestSuite) TestLogoutUser_Success() {

	suite.mockMiddleware.On("DestroySession", mock.Anything, "session-id").Return(nil)
	suite.mockMiddleware.On("CookieDomainForEnv").Return("localhost")
	suite.mockMiddleware.On("CookieSecure").Return(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *AuthHandlerTestSuite) TestLogoutUser_RedisDownStillLogsOut() {

	suite.mockMiddleware.On("DestroySession", mock.Anything, "session-id").
		Return(errors.New("redis unavailable"))
	suite.mockMiddleware.On("CookieDomainForEnv").Return("localhost")
	suite.mockMiddleware.On("CookieSecure").Return(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *AuthHandlerTestSuite) login(returned error) *httptest.ResponseRecorder {
	form := models.LoginForm{
		AuthForm: models.AuthForm{Email: "test@example.com", Password: "password123"},
	}

	suite.mockService.On("ValidateLogin", mock.Anything, form.Email, form.Password, mock.Anything, mock.Anything).
		Return(nil, returned)

	body, _ := json.Marshal(form)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// the handler no longer picks a status, so the sentinel carries it
func (suite *AuthHandlerTestSuite) TestLoginUser_BadCredentialsAreClassified() {
	w := suite.login(services.ErrInvalidCredentials)

	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Invalid email or password", response["message"])
}

func (suite *AuthHandlerTestSuite) TestLoginUser_UnclassifiedDoesNotLeak() {
	w := suite.login(errors.New("pq: relation \"users\" does not exist"))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "pq:")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

// a handler-born error: the body never reaches the service
func (suite *AuthHandlerTestSuite) TestResetPassword_InvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBufferString(`{"password":}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Invalid JSON", response["message"])
}
