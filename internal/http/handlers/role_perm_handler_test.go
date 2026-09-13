package handlers_test

import (
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

type RolePermissionHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockRolePermissionServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.RolePermissionHandler
}

func (suite *RolePermissionHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockRolePermissionServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewRolePermissionHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/roles/:id", suite.handler.GetRoleById)
	suite.router.DELETE("/roles/:id", suite.handler.DeleteRole)
}

func (suite *RolePermissionHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

// a missing role is a 404, even though the repository reports no error for it
func (suite *RolePermissionHandlerTestSuite) TestGetRoleById_NotFound() {
	suite.mockService.On("FetchRoleByID", mock.Anything, int64(7), false).
		Return(nil, services.ErrRoleNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/roles/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Role not found", response["message"])
}

// an unclassified error becomes a generic 500 and the raw cause never reaches the client
func (suite *RolePermissionHandlerTestSuite) TestGetRoleById_UnclassifiedErrorDoesNotLeak() {
	raw := errors.New("pq: relation \"roles\" does not exist")
	suite.mockService.On("FetchRoleByID", mock.Anything, int64(7), false).
		Return(nil, raw)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/roles/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "relation")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

// a handler born error is classified in the handler
func (suite *RolePermissionHandlerTestSuite) TestGetRoleById_BadID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/roles/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("id must be a valid integer", response["message"])
}

// a role still in use is a 409, not a 500
func (suite *RolePermissionHandlerTestSuite) TestDeleteRole_Conflict() {
	suite.mockService.On("DeleteRole", mock.Anything, int64(123), int64(7)).
		Return(services.ErrDefaultRoleDelete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/roles/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Default roles cannot be deleted", response["message"])
}

func TestRolePermissionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RolePermissionHandlerTestSuite))
}
