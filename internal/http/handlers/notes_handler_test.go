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

type NotesHandlerTestSuite struct {
	suite.Suite
	router        *gin.Engine
	mockService   *mocks.MockNotesServiceInterface
	mockValidator *mocks.MockValidator
	handler       *handlers.NotesHandler
}

func (suite *NotesHandlerTestSuite) SetupTest() {

	gin.SetMode(gin.TestMode)

	suite.mockService = mocks.NewMockNotesServiceInterface(suite.T())
	suite.mockValidator = mocks.NewMockValidator(suite.T())

	suite.handler = handlers.NewNotesHandler(
		suite.mockService,
		suite.mockValidator,
	)

	suite.router = gin.New()
	suite.router.Use(middleware.ErrorHandler(zap.NewNop()))

	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Next()
	})

	suite.router.GET("/notes/:id", suite.handler.GetNoteByID)
}

func (suite *NotesHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
}

// a classified service error keeps its status and its message
func (suite *NotesHandlerTestSuite) TestGetNoteByID_NotFound() {
	suite.mockService.On("FetchNoteByID", mock.Anything, int64(123), int64(7)).
		Return(nil, apperr.Wrap(apperr.NotFound, "Note not found", errors.New("record not found")))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notes/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("Note not found", response["message"])
}

// an unclassified error becomes a generic 500 and the raw cause never reaches the client
func (suite *NotesHandlerTestSuite) TestGetNoteByID_UnclassifiedErrorDoesNotLeak() {
	raw := errors.New("pq: relation \"notes\" does not exist")
	suite.mockService.On("FetchNoteByID", mock.Anything, int64(123), int64(7)).
		Return(nil, raw)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notes/7", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.NotContains(w.Body.String(), "relation")

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal(apperr.GenericMessage, response["message"])
}

// a handler born error is classified in the handler
func (suite *NotesHandlerTestSuite) TestGetNoteByID_BadID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notes/abc", nil)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]any
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.Equal("id must be a valid integer", response["message"])
}

func TestNotesHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(NotesHandlerTestSuite))
}
