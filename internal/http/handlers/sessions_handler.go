package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"wealth-warden/internal/models"
	"wealth-warden/internal/sessions"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type SessionsHandler struct {
	store *sessions.Store
	hub   *ws.Hub
}

func NewSessionsHandler(store *sessions.Store, hub *ws.Hub) *SessionsHandler {
	return &SessionsHandler{store: store, hub: hub}
}

func (h *SessionsHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", h.ListSessions)
	apiGroup.DELETE("", h.RevokeAllSessions)
	apiGroup.DELETE("/:id", h.RevokeSession)
}

// The raw session ID never leaves the server; rows are addressed by its hash.
func sessionHandle(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func (h *SessionsHandler) ListSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	currentID, _ := c.Cookie(sessions.CookieName)

	list, err := h.store.ListForUser(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.SessionInfo, 0, len(list))
	for _, s := range list {
		resp = append(resp, models.SessionInfo{
			ID:        sessionHandle(s.ID),
			Device:    utils.DeviceFromUserAgent(s.UserAgent),
			IP:        s.IP,
			CreatedAt: s.CreatedAt,
			LastSeen:  s.LastSeen,
			Current:   s.ID == currentID,
		})
	}
	sort.Slice(resp, func(i, j int) bool {
		if resp[i].Current != resp[j].Current {
			return resp[i].Current
		}
		return resp[i].LastSeen.After(resp[j].LastSeen)
	})

	c.JSON(http.StatusOK, resp)
}

func (h *SessionsHandler) RevokeSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	currentID, _ := c.Cookie(sessions.CookieName)
	handle := c.Param("id")

	list, err := h.store.ListForUser(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	for _, s := range list {
		if sessionHandle(s.ID) != handle {
			continue
		}
		if s.ID == currentID {
			utils.ErrorMessage(c, "Invalid request", "log out to end the current session", http.StatusBadRequest, nil)
			return
		}
		if err := h.store.Delete(ctx, s.ID); err != nil {
			utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
			return
		}
		h.hub.CloseSession(userID, s.ID)
		utils.SuccessMessage(c, "", "Session revoked", http.StatusOK)
		return
	}

	utils.ErrorMessage(c, "Not found", "session not found", http.StatusNotFound, nil)
}

func (h *SessionsHandler) RevokeAllSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.store.DeleteAllForUser(ctx, userID); err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}
	h.hub.CloseUser(userID)

	utils.SuccessMessage(c, "", "Logged out everywhere", http.StatusOK)
}
