package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type authTestEnv struct {
	router *gin.Engine
	store  *sessions.Store
	mr     *miniredis.Miniredis
	wm     *middleware.WebClientMiddleware
}

func newAuthTestEnv(t *testing.T) *authTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cfg, err := config.LoadConfig(nil)
	require.NoError(t, err)

	store := sessions.NewStore(client, cfg.Session)
	wm := middleware.NewWebClientMiddleware(cfg, zap.NewNop(), store)

	router := gin.New()
	router.Use(wm.WebClientAuthentication())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetInt64("user_id")})
	})

	return &authTestEnv{router: router, store: store, mr: mr, wm: wm}
}

func (e *authTestEnv) request(t *testing.T, sessionID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: sessionID})
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func TestWebClientAuthentication_ValidSession(t *testing.T) {
	env := newAuthTestEnv(t)

	id, err := env.store.Create(context.Background(), 123, false, "test-agent", "127.0.0.1")
	require.NoError(t, err)

	w := env.request(t, id)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"user_id":123}`, w.Body.String())
}

func TestWebClientAuthentication_MissingCookie(t *testing.T) {
	env := newAuthTestEnv(t)

	w := env.request(t, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebClientAuthentication_UnknownSession(t *testing.T) {
	env := newAuthTestEnv(t)

	w := env.request(t, "bogus-session-id")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebClientAuthentication_ExpiredSession(t *testing.T) {
	env := newAuthTestEnv(t)

	id, err := env.store.Create(context.Background(), 123, false, "", "")
	require.NoError(t, err)

	env.mr.FastForward(25 * time.Hour)

	w := env.request(t, id)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateLoginSession_MaxAgeMatchesTTL(t *testing.T) {
	env := newAuthTestEnv(t)

	_, maxAge, err := env.wm.CreateLoginSession(context.Background(), 123, false, "", "")
	require.NoError(t, err)
	assert.Equal(t, int((24 * time.Hour).Seconds()), maxAge)

	_, maxAge, err = env.wm.CreateLoginSession(context.Background(), 123, true, "", "")
	require.NoError(t, err)
	assert.Equal(t, int((720 * time.Hour).Seconds()), maxAge)
}

func TestDestroySession(t *testing.T) {
	env := newAuthTestEnv(t)
	ctx := context.Background()

	id, err := env.store.Create(ctx, 123, false, "", "")
	require.NoError(t, err)

	require.NoError(t, env.wm.DestroySession(ctx, id))

	w := env.request(t, id)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
