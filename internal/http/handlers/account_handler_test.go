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

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AccountHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockAccountServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.AccountHandler
}

func (suite *AccountHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockAccountServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewAccountHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()

	// Middleware to inject user_id
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.POST("/accounts", suite.handler.InsertAccount)
}

func (suite *AccountHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

// verifies the path when valid data is provided and the account is created successfully
func (suite *AccountHandlerTestSuite) TestInsertAccount_Success() {

	balance := decimal.NewFromFloat(1000.50)
	payload := &models.AccountReq{
		Name:           "Checking Account",
		Classification: "asset",
		Balance:        &balance,
		AccountTypeID:  1,
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.AccountReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		InsertAccount(
			mock.Anything,
			int64(123),
			mock.MatchedBy(func(req *models.AccountReq) bool {
				return req.Name == "Checking Account" &&
					req.Classification == "asset" &&
					req.Balance.Equal(decimal.NewFromFloat(1000.50)) &&
					req.AccountTypeID == 1
			}),
		).
		Return(nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record created", response["message"])
	suite.Equal("Success", response["title"])
	suite.Equal(float64(200), response["code"])
}

// verifies that malformed JSON returns a 400 status code and doesn't call validator or service
func (suite *AccountHandlerTestSuite) TestInsertAccount_InvalidJSON() {
	invalidJSON := []byte(`{"name": "test", "balance":}`)

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		return
	}

	suite.NotEmpty(response["message"])
	suite.Equal(float64(400), response["code"])

	// Verify that validator and service were never called
	suite.mockValidator.AssertNotCalled(suite.T(), "ValidateStruct")
	suite.mockService.AssertNotCalled(suite.T(), "InsertAccount")
}

// verifies that validation errors return a 422 status code and the service is never called
func (suite *AccountHandlerTestSuite) TestInsertAccount_ValidationFailed() {

	balance := decimal.NewFromFloat(1000.50)
	payload := &models.AccountReq{
		Name:           "",
		Classification: "asset",
		Balance:        &balance,
		AccountTypeID:  0,
	}

	validationErr := errors.New("name is required")

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.AccountReq")).
		Return(validationErr).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)

	// Service should not have been called
	suite.mockService.AssertNotCalled(suite.T(), "InsertAccount")
}

// verifies that service layer errors return a 500 status code
func (suite *AccountHandlerTestSuite) TestInsertAccount_ServiceError() {

	balance := decimal.NewFromFloat(1000.50)
	payload := &models.AccountReq{
		Name:           "Checking Account",
		Classification: "asset",
		Balance:        &balance,
		AccountTypeID:  1,
	}

	serviceErr := errors.New("database connection failed")

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.AccountReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		InsertAccount(mock.Anything, int64(123), mock.Anything).
		Return(serviceErr).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func TestAccountHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountHandlerTestSuite))
}
