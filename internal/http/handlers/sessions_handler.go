package handlers

import (
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type SessionsHandler struct {
	Service services.SessionsServiceInterface
}

func NewSessionsHandler(service services.SessionsServiceInterface) *SessionsHandler {
	return &SessionsHandler{Service: service}
}

func (h *SessionsHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", h.ListSessions)
	apiGroup.DELETE("", h.RevokeAllSessions)
	apiGroup.DELETE("/:id", h.RevokeSession)
}

func (h *SessionsHandler) ListSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	currentID, _ := c.Cookie(sessions.CookieName)

	resp, err := h.Service.ListSessions(ctx, userID, currentID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *SessionsHandler) RevokeSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	currentID, _ := c.Cookie(sessions.CookieName)
	handle := c.Param("id")

	if err := h.Service.RevokeSession(ctx, userID, currentID, handle); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Session revoked", http.StatusOK)
}

func (h *SessionsHandler) RevokeAllSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.Service.RevokeAllSessions(ctx, userID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Logged out everywhere", http.StatusOK)
}
