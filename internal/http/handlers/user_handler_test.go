package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/services"
	"wealth-warden/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type UserHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockUserServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.UserHandler
}

func (suite *UserHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockUserServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewUserHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/users/:id", suite.handler.GetUserById)
	suite.router.GET("/users/token", suite.handler.GetUserByToken)
	suite.router.PUT("/users/:id", suite.handler.UpdateUser)
}

func (suite *UserHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuite) TestGetUserById_NotFound() {
	suite.mockService.On("FetchUserByID", mock.Anything, int64(7)).
		Return(nil, services.ErrUserNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("User not found", response["message"])
}

// an unclassified error becomes a generic 500 and the raw cause never reaches the client
func (suite *UserHandlerTestSuite) TestGetUserById_UnclassifiedErrorDoesNotLeak() {
	raw := errors.New("pq: relation \"users\" does not exist")
	suite.mockService.On("FetchUserByID", mock.Anything, int64(7)).
		Return(nil, raw)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "relation")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

// a dead reset link is a credential problem, not a missing page
func (suite *UserHandlerTestSuite) TestGetUserByToken_Unauthorized() {
	suite.mockService.On("FetchUserByToken", mock.Anything, "reset", "deadbeef").
		Return(nil, services.ErrInvalidLink)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/token?type=reset&value=deadbeef", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("This link is no longer valid", response["message"])
}

// a role the client picked that does not exist is a bad field, not a missing page
func (suite *UserHandlerTestSuite) TestUpdateUser_UnknownRoleIsValidationError() {
	suite.mockValidator.On("ValidateStruct", mock.Anything).Return(nil)
	suite.mockService.On("UpdateUser", mock.Anything, int64(123), int64(7), mock.Anything).
		Return(int64(0), services.ErrInvalidRoleID)

	body, _ := json.Marshal(map[string]any{
		"email":        "someone@example.com",
		"display_name": "Someone",
		"role_id":      999,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/7", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("The selected role does not exist", response["message"])
}

func TestUserHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}
