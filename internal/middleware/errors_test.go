package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/middleware"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func newRequestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.PropagateRequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(middleware.RequestIDKey))
	})
	return r
}

func TestRequestIDUsesTraceID(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	assert.NoError(t, err)

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(trace.ContextWithSpanContext(req.Context(), spanCtx))

	w := httptest.NewRecorder()
	newRequestIDRouter().ServeHTTP(w, req)

	assert.Equal(t, traceID.String(), w.Header().Get(middleware.RequestIDHeader))
	assert.Equal(t, traceID.String(), w.Body.String())
}

func TestRequestIDFallsBackWithoutTrace(t *testing.T) {
	r := newRequestIDRouter()

	call := func() (string, string) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		return w.Header().Get(middleware.RequestIDHeader), w.Body.String()
	}

	header, body := call()
	assert.NotEmpty(t, header)
	assert.Equal(t, header, body)

	other, _ := call()
	assert.NotEqual(t, header, other)
}

func runErrorHandler(t *testing.T, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.ErrorHandler(zap.NewNop()))
	r.GET("/", handler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w
}

func TestErrorHandlerWritesAppError(t *testing.T) {
	w := runErrorHandler(t, func(c *gin.Context) {
		_ = c.Error(apperr.New(apperr.NotFound, "session not found"))
	})

	var body utils.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "session not found", body.Message)
	assert.Equal(t, http.StatusNotFound, body.Code)
}

func TestErrorHandlerHidesUnknownError(t *testing.T) {
	w := runErrorHandler(t, func(c *gin.Context) {
		_ = c.Error(errors.New("pq: duplicate key value violates unique constraint"))
	})

	var body utils.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, apperr.GenericMessage, body.Message)
	assert.NotContains(t, w.Body.String(), "pq:")
}

func TestErrorHandlerKeepsExistingResponse(t *testing.T) {
	w := runErrorHandler(t, func(c *gin.Context) {
		utils.ErrorMessage(c, "Invalid request", "already written", http.StatusBadRequest, errors.New("boom"))
	})

	var body utils.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "already written", body.Message)
}
