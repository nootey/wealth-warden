package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/models"
	"wealth-warden/mocks"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TransactionHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockTransactionServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.TransactionHandler
}

func (suite *TransactionHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockTransactionServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewTransactionHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()

	// Middleware to inject user_id
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.POST("/transactions", suite.handler.InsertTransaction)
	suite.router.POST("/transfers", suite.handler.InsertTransfer)
	suite.router.PUT("/transactions/:id", suite.handler.UpdateTransaction)
	suite.router.DELETE("/transactions/:id", suite.handler.DeleteTransaction)
	suite.router.DELETE("/transfers/:id", suite.handler.DeleteTransfer)
	suite.router.GET("/transactions", suite.handler.GetTransactionsPaginated)
}

func (suite *TransactionHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

func (suite *TransactionHandlerTestSuite) TestInsertTransaction_Success() {
	amount := decimal.NewFromFloat(100.50)
	desc := "Test transaction"
	payload := &models.TransactionReq{
		AccountID:   1,
		Amount:      amount,
		Description: &desc,
		TxnDate:     time.Now(),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransactionReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		InsertTransaction(mock.Anything, int64(123), mock.Anything).
		Return(123, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record created", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestInsertTransaction_ValidationFailed() {
	payload := &models.TransactionReq{
		AccountID: 0,
		TxnDate:   time.Now(),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransactionReq")).
		Return(errors.New("account_id is required")).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "InsertTransaction")
}

func (suite *TransactionHandlerTestSuite) TestInsertTransfer_Success() {
	amount := decimal.NewFromFloat(500.00)
	payload := &models.TransferReq{
		SourceID:      1,
		DestinationID: 2,
		Amount:        amount,
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransferReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		InsertTransfer(mock.Anything, int64(123), mock.Anything).
		Return(123, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record created", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestInsertTransfer_ValidationFailed() {
	payload := &models.TransferReq{
		SourceID:      0,
		DestinationID: 0,
		Amount:        decimal.NewFromInt(0),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransferReq")).
		Return(errors.New("from_account_id is required")).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "InsertTransfer")
}

func (suite *TransactionHandlerTestSuite) TestUpdateTransaction_Success() {
	amount := decimal.NewFromFloat(200.00)
	desc := "Updated transaction"
	payload := &models.TransactionReq{
		AccountID:   1,
		Amount:      amount,
		Description: &desc,
		TxnDate:     time.Now(),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransactionReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		UpdateTransaction(mock.Anything, int64(123), int64(1), mock.Anything).
		Return(123, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/transactions/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record updated", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestUpdateTransaction_ValidationFailed() {
	payload := &models.TransactionReq{
		AccountID: 0,
		TxnDate:   time.Now(),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("*models.TransactionReq")).
		Return(errors.New("account_id is required")).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/transactions/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "UpdateTransaction")
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction_Success() {
	suite.mockService.EXPECT().
		DeleteTransaction(mock.Anything, int64(123), int64(1)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/1", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record deleted", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction_ServiceError() {
	suite.mockService.EXPECT().
		DeleteTransaction(mock.Anything, int64(123), int64(999)).
		Return(errors.New("transaction not found")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/999", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransfer_Success() {
	suite.mockService.EXPECT().
		DeleteTransfer(mock.Anything, int64(123), int64(1)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/transfers/1", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Record deleted", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestGetTransactionsPaginated_Success() {
	desc := "Test transaction"
	mockTransactions := []models.Transaction{
		{
			ID:          1,
			AccountID:   1,
			Description: &desc,
			UserID:      123,
		},
	}

	mockPaginator := &utils.Paginator{
		CurrentPage:  1,
		RowsPerPage:  10,
		From:         1,
		To:           1,
		TotalRecords: 1,
	}

	suite.mockService.EXPECT().
		FetchTransactionsPaginated(
			mock.Anything,
			int64(123),
			mock.Anything,
			false,
			mock.Anything,
		).
		Return(mockTransactions, mockPaginator, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/transactions?page=1&rowsPerPage=10", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(float64(1), response["current_page"])
	suite.Equal(float64(1), response["total_records"])
	suite.NotNil(response["data"])
}

func (suite *TransactionHandlerTestSuite) TestGetTransactionsPaginated_ServiceError() {
	suite.mockService.EXPECT().
		FetchTransactionsPaginated(
			mock.Anything,
			int64(123),
			mock.Anything,
			false,
			mock.Anything,
		).
		Return(nil, nil, errors.New("database error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/transactions?page=1&rows=10", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func TestTransactionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionHandlerTestSuite))
}
