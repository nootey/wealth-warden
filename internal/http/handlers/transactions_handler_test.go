package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/mocks"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
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
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	// Middleware to inject user_id
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.POST("/transactions", suite.handler.InsertTransaction)
	suite.router.PUT("/transactions/bulk", suite.handler.BulkInsertTransactions)
	suite.router.POST("/transfers", suite.handler.InsertTransfer)
	suite.router.PUT("/transactions/:id", suite.handler.UpdateTransaction)
	suite.router.DELETE("/transactions/:id", suite.handler.DeleteTransaction)
	suite.router.DELETE("/transfers/:id", suite.handler.DeleteTransfer)
	suite.router.GET("/transactions", suite.handler.GetTransactionsPaginated)
	suite.router.POST("/transactions/restore", suite.handler.RestoreTransaction)
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
		Return(models.InsertResult{ID: 123}, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
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

func (suite *TransactionHandlerTestSuite) TestBulkInsertTransactions_Success() {
	payload := &models.BulkInsertTransactionReq{
		AccountID: 1,
		TxnDate:   time.Now(),
		Rows: []models.BulkInsertRowReq{
			{Direction: "expense", Amount: decimal.NewFromInt(100)},
			{Direction: "income", Amount: decimal.NewFromInt(200)},
		},
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("models.BulkInsertTransactionReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		BulkInsertTransactions(mock.Anything, int64(123), mock.Anything).
		Return(&models.BulkTransactionResult{Processed: 2}, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/transactions/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	suite.Require().NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("2 transaction(s) created", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestBulkInsertTransactions_AllDuplicates() {
	payload := &models.BulkInsertTransactionReq{
		AccountID: 1,
		TxnDate:   time.Now(),
		Rows:      []models.BulkInsertRowReq{{Direction: "expense", Amount: decimal.NewFromInt(100)}},
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("models.BulkInsertTransactionReq")).
		Return(nil).
		Once()

	suite.mockService.EXPECT().
		BulkInsertTransactions(mock.Anything, int64(123), mock.Anything).
		Return(&models.BulkTransactionResult{Skipped: 1}, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/transactions/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusAlreadyReported, w.Code)
}

func (suite *TransactionHandlerTestSuite) TestBulkInsertTransactions_ValidationFailed() {
	payload := &models.BulkInsertTransactionReq{
		AccountID: 1,
		TxnDate:   time.Now(),
	}

	suite.mockValidator.EXPECT().
		ValidateStruct(mock.AnythingOfType("models.BulkInsertTransactionReq")).
		Return(errors.New("rows is required")).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/transactions/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "BulkInsertTransactions")
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
		Return(models.InsertResult{ID: 123}, nil).
		Once()

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
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
	suite.Require().NoError(err)
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
	suite.Require().NoError(err)
	suite.Equal("Record deleted", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction_ServiceError() {
	suite.mockService.EXPECT().
		DeleteTransaction(mock.Anything, int64(123), int64(999)).
		Return(errors.New("some raw internal database error")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/999", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(apperr.GenericMessage, response["message"])
	suite.NotContains(w.Body.String(), "database error")
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction_ClassifiedError() {
	suite.mockService.EXPECT().
		DeleteTransaction(mock.Anything, int64(123), int64(999)).
		Return(services.ErrTransactionNotFound).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/999", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal("Transaction not found", response["message"])
}

func (suite *TransactionHandlerTestSuite) TestDeleteTransaction_InvalidID() {
	req := httptest.NewRequest(http.MethodDelete, "/transactions/not-a-number", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "DeleteTransaction")
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
	suite.Require().NoError(err)
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

	next := "next-page"
	mockPaginator := &utils.Paginator{
		RowsPerPage:  10,
		TotalRecords: 1,
		NextCursor:   &next,
	}

	mockTotals := models.TransactionBatchTotals{
		Count:    1,
		Income:   decimal.NewFromInt(0),
		Expenses: decimal.NewFromInt(0),
	}

	suite.mockService.EXPECT().
		FetchTransactionsPaginated(
			mock.Anything,
			int64(123),
			mock.MatchedBy(func(p utils.PaginationParams) bool { return p.Cursor == "this-page" }),
			false,
			mock.Anything,
		).
		Return(mockTransactions, &mockTotals, mockPaginator, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/transactions?cursor=this-page&rowsPerPage=10", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.EqualValues(1, response["total_records"])
	suite.Equal("next-page", response["next_cursor"])
	suite.Nil(response["prev_cursor"])
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
		Return(nil, nil, nil, errors.New("database error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/transactions?rowsPerPage=10", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// the restore body was bound but never validated, so a missing id passed as 0
func (suite *TransactionHandlerTestSuite) TestRestoreTransaction_ValidationFails() {
	suite.mockValidator.EXPECT().
		ValidateStruct(mock.Anything).
		Return(errors.New("id is required")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/transactions/restore", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.mockService.AssertNotCalled(suite.T(), "RestoreTransaction")
}

func TestTransactionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionHandlerTestSuite))
}
