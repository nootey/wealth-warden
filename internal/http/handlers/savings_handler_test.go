package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

type SavingsHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockSavingsServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.SavingsHandler
}

func (suite *SavingsHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockSavingsServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewSavingsHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/savings/:id", suite.handler.GetGoalByID)
	suite.router.POST("/savings/:id/fund", suite.handler.FundGoalNow)
	suite.router.PUT("/savings/:id/contributions", suite.handler.InsertContribution)
}

func (suite *SavingsHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

// a missing goal used to answer 500 with the raw GORM text
func (suite *SavingsHandlerTestSuite) TestGetGoalByID_NotFound() {
	suite.mockService.On("FetchGoalByID", mock.Anything, int64(123), int64(7)).
		Return(nil, services.ErrGoalNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/savings/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Goal not found", response["message"])
}

func (suite *SavingsHandlerTestSuite) TestGetGoalByID_UnclassifiedDoesNotLeak() {
	suite.mockService.On("FetchGoalByID", mock.Anything, int64(123), int64(7)).
		Return(nil, errors.New("pq: relation \"saving_goals\" does not exist"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/savings/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "saving_goals")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

func (suite *SavingsHandlerTestSuite) TestGetGoalByID_BadID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/savings/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("id must be a valid integer", response["message"])
}

// a goal the user cannot fund twice is a state conflict, not a 500
func (suite *SavingsHandlerTestSuite) TestFundGoalNow_AlreadyFunded() {
	suite.mockService.On("FundGoalNow", mock.Anything, int64(123), int64(7)).
		Return(apperr.New(apperr.Conflict, "goal already funded this month"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/savings/7/fund", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("goal already funded this month", response["message"])
}

// the balance cap is the message a user acts on, so it must survive the middleware
func (suite *SavingsHandlerTestSuite) TestInsertContribution_ExceedsBalance() {
	suite.mockValidator.On("ValidateStruct", mock.Anything).Return(nil)
	suite.mockService.On("InsertContribution", mock.Anything, int64(123), int64(7), mock.Anything).
		Return(int64(0), apperr.New(apperr.Validation, "contribution of 50.00 exceeds uncategorized balance of 10.00"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/savings/7/contributions",
		strings.NewReader(`{"amount":"50","month":"2026-09-01"}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("contribution of 50.00 exceeds uncategorized balance of 10.00", response["message"])
}

func (suite *SavingsHandlerTestSuite) TestInsertContribution_BadJSON() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/savings/7/contributions", strings.NewReader(`{"amount":`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Invalid JSON", response["message"])
}

func TestSavingsHandlerSuite(t *testing.T) {
	suite.Run(t, new(SavingsHandlerTestSuite))
}
