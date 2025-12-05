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
	suite.router.GET("/accounts/:id", suite.handler.GetAccountByID)
	suite.router.PUT("/accounts/:id", suite.handler.UpdateAccount)
	suite.router.DELETE("/accounts/:id", suite.handler.CloseAccount)
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
		Return(123, nil).
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
		Return(0, serviceErr).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// verifies getting an account by ID with valid ID returns the account
func (suite *AccountHandlerTestSuite) TestGetAccountByID_Success() {
	mockAccount := &models.Account{
		ID:            1,
		Name:          "Savings Account",
		AccountTypeID: 2,
		UserID:        123,
	}

	suite.mockService.EXPECT().
		FetchAccountByID(mock.Anything, int64(123), int64(1), false).
		Return(mockAccount, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response models.Account
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(int64(1), response.ID)
	suite.Equal("Savings Account", response.Name)
	suite.Equal(int64(123), response.UserID)
}

// verifies that a non-existent account returns appropriate error
func (suite *AccountHandlerTestSuite) TestGetAccountByID_NotFound() {
	suite.mockService.EXPECT().
		FetchAccountByID(mock.Anything, int64(123), int64(999), false).
		Return(nil, errors.New("account not found")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/accounts/999", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// verifies updating an account with valid data succeeds
func (suite *AccountHandlerTestSuite) TestUpdateAccount_Success() {
	balance := decimal.NewFromFloat(2000.00)
	payload := &models.AccountReq{
		Name:           "Updated Account",
		Classification: "asset",
		Balance:        &balance,
		AccountTypeID:  1,
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.AccountReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		UpdateAccount(mock.Anything, int64(123), int64(1), mock.Anything).
		Return(123, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/accounts/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record updated", response["message"])
}

// verifies closing an account succeeds
func (suite *AccountHandlerTestSuite) TestCloseAccount_Success() {
	suite.mockService.EXPECT().
		CloseAccount(mock.Anything, int64(123), int64(1)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/accounts/1", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Success", response["title"])
}

// verifies closing a non-existent account returns error
func (suite *AccountHandlerTestSuite) TestCloseAccount_NotFound() {
	suite.mockService.EXPECT().
		CloseAccount(mock.Anything, int64(123), int64(999)).
		Return(errors.New("account not found")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/accounts/999", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func TestAccountHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountHandlerTestSuite))
}
