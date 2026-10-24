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
	name string
	err  error
}

func newStub(name string, err error) *stub {
	return &stub{name: name, err: err}
}

func (s *stub) Name() string                  { return s.name }
func (s *stub) Check(_ context.Context) error { return s.err }

// waitReported polls the handler until it reports every named check.
// Check returns before Run stores the result, so a signal from the stub races the handler.
func waitReported(t *testing.T, svc *health.Service, names ...string) {
	t.Helper()
	require.Eventually(t, func() bool {
		rec := httptest.NewRecorder()
		svc.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		var body struct {
			Checks map[string]any `json:"checks"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			return false
		}
		for _, name := range names {
			if _, ok := body.Checks[name]; !ok {
				return false
			}
		}
		return true
	}, time.Second, 5*time.Millisecond, "checks %v not reported within 1s", names)
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
	waitReported(t, svc, "db")

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
	waitReported(t, svc, "db")

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
	waitReported(t, svc, "db", "kafka")

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
