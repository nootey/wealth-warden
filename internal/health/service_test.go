package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wealth-warden/internal/health"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stub struct {
	name    string
	err     error
	checked chan struct{}
}

func newStub(name string, err error) *stub {
	return &stub{name: name, err: err, checked: make(chan struct{}, 1)}
}

func (s *stub) Name() string { return s.name }
func (s *stub) Check(_ context.Context) error {
	select {
	case s.checked <- struct{}{}:
	default:
	}
	return s.err
}

func (s *stub) waitChecked(t *testing.T) {
	t.Helper()
	select {
	case <-s.checked:
	case <-time.After(time.Second):
		t.Fatal("check not called within 1s")
	}
}

func newSvc(t *testing.T) *health.Service {
	t.Helper()
	svc, err := health.New(zap.NewNop())
	require.NoError(t, err)
	return svc
}

func startSvc(t *testing.T, svc *health.Service) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go svc.Run(ctx)
}

// TestHandler_AllHealthy verifies 200 + ok status when all checks pass.
func TestHandler_AllHealthy(t *testing.T) {
	svc := newSvc(t)
	db := newStub("db", nil)
	svc.Add(db)
	startSvc(t, svc)
	db.waitChecked(t)

	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
	dbCheck := body["checks"].(map[string]any)["db"].(map[string]any)
	assert.Equal(t, "ok", dbCheck["status"])
	assert.Empty(t, dbCheck["error"])
}

// TestHandler_Degraded verifies 503 + error details when a check fails.
func TestHandler_Degraded(t *testing.T) {
	svc := newSvc(t)
	db := newStub("db", errors.New("connection refused"))
	svc.Add(db)
	startSvc(t, svc)
	db.waitChecked(t)

	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "degraded", body["status"])
	dbCheck := body["checks"].(map[string]any)["db"].(map[string]any)
	assert.Equal(t, "error", dbCheck["status"])
	assert.Equal(t, "connection refused", dbCheck["error"])
}

// TestHandler_PartialFailure verifies 503 when any single check fails.
func TestHandler_PartialFailure(t *testing.T) {
	svc := newSvc(t)
	db := newStub("db", nil)
	kafka := newStub("kafka", errors.New("broker unavailable"))
	svc.Add(db)
	svc.Add(kafka)
	startSvc(t, svc)
	db.waitChecked(t)
	kafka.waitChecked(t)

	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "degraded", body["status"])
	checks := body["checks"].(map[string]any)
	assert.Equal(t, "ok", checks["db"].(map[string]any)["status"])
	assert.Equal(t, "error", checks["kafka"].(map[string]any)["status"])
}
