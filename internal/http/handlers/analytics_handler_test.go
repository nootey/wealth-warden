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
	"wealth-warden/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type AnalyticsHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *mocks.MockAnalyticsServiceInterface
	handler     *handlers.AnalyticsHandler
}

func (suite *AnalyticsHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockAnalyticsServiceInterface(suite.T())
	suite.handler = handlers.NewAnalyticsHandler(suite.mockService)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/reports/:id/download", suite.handler.DownloadReport)
	suite.router.GET("/networth", suite.handler.NetWorthChart)
	suite.router.GET("/cashflow", suite.handler.GetYearlyCashFlowBreakdown)
}

func (suite *AnalyticsHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *AnalyticsHandlerTestSuite) download(returned error) *httptest.ResponseRecorder {
	suite.mockService.On("DownloadReport", mock.Anything, int64(7), int64(123)).
		Return(nil, "", returned)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/7/download", nil)
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *AnalyticsHandlerTestSuite) TestDownloadReport_NotReady() {
	w := suite.download(apperr.New(apperr.Conflict, "report is not ready for download"))

	suite.Equal(http.StatusConflict, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("report is not ready for download", response["message"])
}

// the file path used to reach the client through err.Error()
func (suite *AnalyticsHandlerTestSuite) TestDownloadReport_MissingFileHidesPath() {
	w := suite.download(errors.New("open /srv/wealth-warden/reports/7.xlsx: no such file or directory"))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "/srv/wealth-warden")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

func (suite *AnalyticsHandlerTestSuite) TestNetWorthChart_BadAccountParam() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/networth?account=abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("account must be a valid integer", response["message"])
}

// an unvalidated year used to fall through as 0 and return an empty chart
func (suite *AnalyticsHandlerTestSuite) TestGetYearlyCashFlowBreakdown_YearIsChecked() {
	cases := map[string]string{
		"missing": "/cashflow",
		"garbage": "/cashflow?year=abc",
		"range":   "/cashflow?year=12",
	}

	for name, url := range cases {
		suite.Run(name, func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			suite.router.ServeHTTP(w, req)

			suite.Equal(http.StatusBadRequest, w.Code)
		})
	}
}

func TestAnalyticsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsHandlerTestSuite))
}
