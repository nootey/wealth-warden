package middleware

import (
	"context"
	"errors"
	"net/http"
	"wealth-warden/internal/config"
	"wealth-warden/internal/sessions"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type WebClientMiddlewareInterface interface {
	CookieDomainForEnv() string
	CookieSecure() bool
	WebClientAuthentication() gin.HandlerFunc
	CreateLoginSession(ctx context.Context, userID int64, rememberMe bool, userAgent, ip string) (string, int, error)
	DestroySession(ctx context.Context, sessionID string) error
}

var _ WebClientMiddlewareInterface = (*WebClientMiddleware)(nil)

type WebClientMiddleware struct {
	config   *config.Config
	logger   *zap.Logger
	sessions *sessions.Store
}

func NewWebClientMiddleware(cfg *config.Config, logger *zap.Logger, store *sessions.Store) *WebClientMiddleware {
	return &WebClientMiddleware{
		config:   cfg,
		logger:   logger,
		sessions: store,
	}
}

func (m *WebClientMiddleware) CookieDomainForEnv() string {
	cfg := m.config
	if !cfg.Release {
		return ""
	}
	return cfg.WebClient.Domain
}

func (m *WebClientMiddleware) CookieSecure() bool {
	return m.config.Release
}

func (m *WebClientMiddleware) WebClientAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := c.Cookie(sessions.CookieName)
		if err != nil || id == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		userID, err := m.sessions.Validate(c.Request.Context(), id)
		if err != nil {
			if !errors.Is(err, sessions.ErrNotFound) {
				m.logger.Error("session validation failed", zap.Error(err))
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func (m *WebClientMiddleware) CreateLoginSession(ctx context.Context, userID int64, rememberMe bool, userAgent, ip string) (string, int, error) {
	id, err := m.sessions.Create(ctx, userID, rememberMe, userAgent, ip)
	if err != nil {
		return "", 0, err
	}
	return id, int(m.sessions.TTL(rememberMe).Seconds()), nil
}

func (m *WebClientMiddleware) DestroySession(ctx context.Context, sessionID string) error {
	return m.sessions.Delete(ctx, sessionID)
}
