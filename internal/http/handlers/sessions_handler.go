package handlers

import (
	"errors"
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
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *SessionsHandler) RevokeSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	currentID, _ := c.Cookie(sessions.CookieName)
	handle := c.Param("id")

	err := h.Service.RevokeSession(ctx, userID, currentID, handle)
	switch {
	case err == nil:
		utils.SuccessMessage(c, "", "Session revoked", http.StatusOK)
	case errors.Is(err, services.ErrCannotRevokeCurrentSession):
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
	case errors.Is(err, sessions.ErrNotFound):
		utils.ErrorMessage(c, "Not found", "session not found", http.StatusNotFound, nil)
	default:
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
	}
}

func (h *SessionsHandler) RevokeAllSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.Service.RevokeAllSessions(ctx, userID); err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "", "Logged out everywhere", http.StatusOK)
}
