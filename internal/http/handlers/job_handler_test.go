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
	"wealth-warden/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type JobHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *mocks.MockJobServiceInterface
	handler     *handlers.JobHandler
}

func (suite *JobHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockJobServiceInterface(suite.T())
	suite.handler = handlers.NewJobHandler(suite.mockService)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/jobs/:id", suite.handler.GetJob)
	suite.router.POST("/jobs/delete", suite.handler.DeleteJobs)
}

func (suite *JobHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *JobHandlerTestSuite) getJob(returned error) *httptest.ResponseRecorder {
	suite.mockService.On("FetchJob", mock.Anything, int64(42)).Return(nil, returned)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/jobs/42", nil)
	suite.router.ServeHTTP(w, req)
	return w
}

// the handler used to run errors.Is itself; the status now rides on the error
func (suite *JobHandlerTestSuite) TestGetJob_NotFound() {
	w := suite.getJob(apperr.Wrap(apperr.NotFound, "job not found", errors.New("river: not found")))

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("job not found", response["message"])
}

func (suite *JobHandlerTestSuite) TestGetJob_UnclassifiedDoesNotLeak() {
	w := suite.getJob(errors.New("pq: relation \"river_job\" does not exist"))

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "river_job")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

func (suite *JobHandlerTestSuite) TestGetJob_BadID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/jobs/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("id must be a valid integer", response["message"])
}

// a running job cannot be deleted, and River says so through its own sentinel
func (suite *JobHandlerTestSuite) TestDeleteJobs_Running() {
	suite.mockService.On("DeleteJobs", mock.Anything, []int64{7}).
		Return(apperr.Wrap(apperr.Conflict, "job is running and cannot be changed", errors.New("job running")))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/jobs/delete", strings.NewReader(`{"ids":[7]}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("job is running and cannot be changed", response["message"])
}

// the empty-id guard moved into the service, so the handler must forward the body untouched
func (suite *JobHandlerTestSuite) TestDeleteJobs_EmptyIDsReachesService() {
	suite.mockService.On("DeleteJobs", mock.Anything, []int64{}).
		Return(apperr.New(apperr.Invalid, "no job ids provided"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/jobs/delete", strings.NewReader(`{"ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("no job ids provided", response["message"])
}

func TestJobHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(JobHandlerTestSuite))
}
