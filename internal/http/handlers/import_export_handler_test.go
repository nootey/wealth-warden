package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"wealth-warden/internal/apperr"
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

func newErrorRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(zap.NewNop()))
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})
	return r
}

func decode(t interface {
	NoError(error, ...interface{}) bool
}, w *httptest.ResponseRecorder) map[string]any {
	var response map[string]any
	t.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	return response
}

// --- export ---

type ExportHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *mocks.MockExportServiceInterface
}

func TestExportHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ExportHandlerTestSuite))
}

func (suite *ExportHandlerTestSuite) SetupTest() {
	suite.mockService = mocks.NewMockExportServiceInterface(suite.T())
	handler := handlers.NewExportHandler(suite.mockService, nil)

	suite.router = newErrorRouter()
	suite.router.POST("/exports/:id/download", handler.DownloadExport)
	suite.router.DELETE("/exports/:id", handler.DeleteExport)
}

func (suite *ExportHandlerTestSuite) download(returned error) *httptest.ResponseRecorder {
	suite.mockService.On("DownloadExport", mock.Anything, int64(7), int64(123)).
		Return(nil, returned)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/exports/7/download", nil)
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *ExportHandlerTestSuite) TestDownload_NotReadyIsClassified() {
	w := suite.download(services.ErrExportNotReady)

	suite.Equal(http.StatusConflict, w.Code)
	suite.Equal("That export is not ready to download yet", decode(suite, w)["message"])
}

// before the conversion this reported "id must be a valid integer" with a 400
func (suite *ExportHandlerTestSuite) TestDownload_MissingExportIs404() {
	w := suite.download(services.ErrExportNotFound)

	suite.Equal(http.StatusNotFound, w.Code)
	suite.Equal("Export not found", decode(suite, w)["message"])
}

func (suite *ExportHandlerTestSuite) TestDownload_UnclassifiedDoesNotLeak() {
	w := suite.download(errors.New("open /storage/exports/123/x.zip: permission denied"))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "/storage/")
	suite.Equal(apperr.GenericMessage, decode(suite, w)["message"])
}

// a handler-born error: the id never reaches the service
func (suite *ExportHandlerTestSuite) TestDelete_BadIDNeverReachesService() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/exports/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
	suite.Equal("id must be a valid integer", decode(suite, w)["message"])
	suite.mockService.AssertNotCalled(suite.T(), "DeleteExport", mock.Anything, mock.Anything, mock.Anything)
}

// --- import ---

type ImportHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *mocks.MockImportServiceInterface
}

func TestImportHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ImportHandlerTestSuite))
}

func (suite *ImportHandlerTestSuite) SetupTest() {
	suite.mockService = mocks.NewMockImportServiceInterface(suite.T())
	handler := handlers.NewImportHandler(suite.mockService, nil)

	suite.router = newErrorRouter()
	suite.router.POST("/imports/validate", handler.ValidateCustomImport)
	suite.router.POST("/imports/bank/transactions", handler.ImportBankTransactions)
}

// bankImport posts one statement file with the given extra form fields; the parser is mocked to return two rows.
func (suite *ImportHandlerTestSuite) bankImport(fields map[string]string) *httptest.ResponseRecorder {
	tx1, tx2 := "TX1", "TX2"
	suite.mockService.On("ParseBankStatements", "nlb", mock.Anything).
		Return(models.TxnImportPayload{Identifier: "nlb_2017-01", Txns: []models.JSONTxn{
			{Amount: "1.00", ExternalTxnID: &tx1},
			{Amount: "2.00", ExternalTxnID: &tx2},
		}}, nil).Once()

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("files", "statement.csv")
	_, _ = fw.Write([]byte("x"))
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/imports/bank/transactions?check_acc_id=1", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *ImportHandlerTestSuite) TestBankImport_SkipsRowsAfterSettingCategories() {
	suite.mockService.On("ImportTransactions", mock.Anything, int64(123), int64(1), models.ImportTypeBank, mock.MatchedBy(func(p models.TxnImportPayload) bool {
		return len(p.Txns) == 1 && *p.Txns[0].ExternalTxnID == "TX2" && p.Txns[0].CategoryID != nil && *p.Txns[0].CategoryID == 5
	})).Return(0, nil).Once()

	w := suite.bankImport(map[string]string{
		"row_categories": `[{"row":1,"category_id":5}]`,
		"skip_rows":      `[0]`,
	})

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *ImportHandlerTestSuite) TestBankImport_AllRowsSkippedNeverReachesService() {
	w := suite.bankImport(map[string]string{"skip_rows": `[0,1]`})

	suite.Equal(http.StatusBadRequest, w.Code)
	suite.Contains(decode(suite, w)["message"], "nothing to import")
}

func (suite *ImportHandlerTestSuite) validate(returned error) *httptest.ResponseRecorder {
	suite.mockService.On("ValidateCustomImport", mock.Anything, mock.Anything, "cash").
		Return(nil, 0, returned)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/imports/validate?step=cash",
		strings.NewReader(`{"identifier":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)
	return w
}

// the row-level detail has to survive: it is the whole point of the import screen
func (suite *ImportHandlerTestSuite) TestValidate_RowErrorKeepsItsDetail() {
	w := suite.validate(apperr.New(apperr.Validation, "A row is missing its amount"))

	suite.Equal(http.StatusUnprocessableEntity, w.Code)
	suite.Equal("A row is missing its amount", decode(suite, w)["message"])
}

func (suite *ImportHandlerTestSuite) TestValidate_DuplicateImportIs409() {
	w := suite.validate(services.ErrImportFileExists)

	suite.Equal(http.StatusConflict, w.Code)
	suite.Equal("An import with that name already exists", decode(suite, w)["message"])
}

func (suite *ImportHandlerTestSuite) TestValidate_UnclassifiedDoesNotLeak() {
	w := suite.validate(errors.New(`pq: duplicate key value violates unique constraint "imports_pkey"`))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "pq:")
	suite.Equal(apperr.GenericMessage, decode(suite, w)["message"])
}

func (suite *ImportHandlerTestSuite) TestValidate_BadJSONNeverReachesService() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/imports/validate", strings.NewReader(`{"identifier":}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
	suite.Equal("Invalid JSON", decode(suite, w)["message"])
	suite.mockService.AssertNotCalled(suite.T(), "ValidateCustomImport",
		mock.Anything, mock.Anything, mock.Anything)
}
