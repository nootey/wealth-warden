package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/models"
	"wealth-warden/mocks"
	"wealth-warden/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.POST("/auth/login", suite.handler.LoginUser)
	suite.router.POST("/auth/signup", suite.handler.SignUp)
	suite.router.GET("/auth/user", suite.handler.GetAuthUser)
	suite.router.POST("/auth/logout", suite.handler.LogoutUser)
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

	suite.mockMiddleware.On("GenerateLoginTokens", user.ID, form.RememberMe).
		Return("access_token", "refresh_token", nil)
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

func (suite *AuthHandlerTestSuite) TestLoginUser_TokenGenerationFails() {
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

	suite.mockMiddleware.On("GenerateLoginTokens", user.ID, form.RememberMe).
		Return("", "", errors.New("token generation failed"))

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

	suite.mockMiddleware.On("CookieDomainForEnv").Return("localhost")
	suite.mockMiddleware.On("CookieSecure").Return(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}
