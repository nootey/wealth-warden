package middleware

import (
	"context"
	"errors"
	"net/http"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type WebClientMiddlewareInterface interface {
	CookieDomainForEnv() string
	CookieSecure() bool
	WebClientAuthentication() gin.HandlerFunc
	CreateLoginSession(ctx context.Context, userID int64, rememberMe bool, userAgent, ip string) (string, int, error)
	DestroySession(ctx context.Context, sessionID string) error
	ErrorLogger() gin.HandlerFunc
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

func (m *WebClientMiddleware) ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process request

		// After request
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				m.logger.Info("HTTP error",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("client_ip", c.ClientIP()),
					zap.Int("status_code", c.Writer.Status()),
					zap.Error(err),
				)
			}
		}
	}
}
