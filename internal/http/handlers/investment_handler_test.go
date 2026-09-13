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

type InvestmentHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockInvestmentServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.InvestmentHandler
}

func (suite *InvestmentHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockInvestmentServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewInvestmentHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/investments/:id", suite.handler.GetInvestmentAssetByID)
	suite.router.PUT("/investments/trades", suite.handler.InsertInvestmentTrade)
	suite.router.DELETE("/investments/tax-brackets/:id", suite.handler.DeleteTaxBracket)
	suite.router.POST("/investments/tax-brackets/copy", suite.handler.CopyTaxBrackets)
}

func (suite *InvestmentHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

// this read used to answer 400 with the raw GORM text for a missing asset
func (suite *InvestmentHandlerTestSuite) TestGetAssetByID_NotFound() {
	suite.mockService.On("FetchInvestmentAssetByID", mock.Anything, int64(123), int64(7)).
		Return(nil, services.ErrAssetNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/investments/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Asset not found", response["message"])
}

// the same read answered 400 on a database failure, and sent the driver text out
func (suite *InvestmentHandlerTestSuite) TestGetAssetByID_UnclassifiedDoesNotLeak() {
	suite.mockService.On("FetchInvestmentAssetByID", mock.Anything, int64(123), int64(7)).
		Return(nil, errors.New("pq: relation \"investment_assets\" does not exist"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/investments/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "investment_assets")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

func (suite *InvestmentHandlerTestSuite) TestGetAssetByID_BadID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/investments/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("id must be a valid integer", response["message"])
}

// the amounts in this message are the point of it, so it must reach the client intact
func (suite *InvestmentHandlerTestSuite) TestInsertTrade_InsufficientFunds() {
	suite.mockValidator.On("ValidateStruct", mock.Anything).Return(nil)
	suite.mockService.On("InsertInvestmentTrade", mock.Anything, int64(123), mock.Anything).
		Return(int64(0), apperr.New(apperr.Validation, "insufficient funds: need 500.00 EUR but only 120.00 EUR available"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/investments/trades",
		strings.NewReader(`{"asset_id":1,"trade_type":"buy","quantity":"5","price_per_unit":"100"}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("insufficient funds: need 500.00 EUR but only 120.00 EUR available", response["message"])
}

// deleting a bracket that is not there used to answer 200 "Record deleted"
func (suite *InvestmentHandlerTestSuite) TestDeleteTaxBracket_NotFound() {
	suite.mockService.On("DeleteTaxBracket", mock.Anything, int64(123), int64(9)).
		Return(services.ErrTaxBracketNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/investments/tax-brackets/9", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Tax bracket not found", response["message"])
}

func (suite *InvestmentHandlerTestSuite) TestCopyTaxBrackets_Conflict() {
	suite.mockValidator.On("ValidateStruct", mock.Anything).Return(nil)
	suite.mockService.On("CopyTaxBrackets", mock.Anything, int64(123), mock.Anything, mock.Anything).
		Return(apperr.New(apperr.Conflict, "crypto already has brackets configured"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/investments/tax-brackets/copy",
		strings.NewReader(`{"from_type":"stock","to_type":"crypto"}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("crypto already has brackets configured", response["message"])
}

func TestInvestmentHandlerSuite(t *testing.T) {
	suite.Run(t, new(InvestmentHandlerTestSuite))
}
