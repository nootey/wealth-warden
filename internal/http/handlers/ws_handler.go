package handlers

import (
	"net/url"
	"strings"
	"wealth-warden/internal/sessions"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/config"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

type WebsocketHandler struct {
	hub  *ws.Hub
	opts *websocket.AcceptOptions
}

func NewWebsocketHandler(hub *ws.Hub, cfg *config.Config) *WebsocketHandler {
	return &WebsocketHandler{hub: hub, opts: acceptOptions(cfg)}
}

func (h *WebsocketHandler) Routes(rg *gin.RouterGroup) {
	rg.GET("", h.connect)
}

func (h *WebsocketHandler) connect(c *gin.Context) {
	userID := c.GetInt64("user_id")
	sessionID, _ := c.Cookie(sessions.CookieName)

	conn, err := websocket.Accept(c.Writer, c.Request, h.opts)
	if err != nil {
		_ = c.Error(err)
		return
	}

	h.hub.Serve(c.Request.Context(), ws.NewClient(userID, sessionID, conn))
}

// Websockets bypass CORS, so the origin is checked here instead.
func acceptOptions(cfg *config.Config) *websocket.AcceptOptions {
	if !cfg.Release {
		return &websocket.AcceptOptions{InsecureSkipVerify: true}
	}

	patterns := make([]string, 0, len(cfg.CORS.AllowedOrigins)+len(cfg.CORS.WildcardSuffixes))
	for _, origin := range cfg.CORS.AllowedOrigins {
		if u, err := url.Parse(strings.TrimSpace(origin)); err == nil && u.Host != "" {
			patterns = append(patterns, u.Host)
		}
	}
	for _, suffix := range cfg.CORS.WildcardSuffixes {
		if suffix = strings.TrimSpace(suffix); suffix != "" {
			patterns = append(patterns, "*"+strings.ToLower(suffix))
		}
	}
	return &websocket.AcceptOptions{OriginPatterns: patterns}
}
