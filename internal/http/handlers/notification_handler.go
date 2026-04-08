package handlers

import (
	"net/http"
	"strconv"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service services.NotificationServiceInterface
}

func NewNotificationHandler(service services.NotificationServiceInterface) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) Routes(rg *gin.RouterGroup) {
	rg.GET("", h.GetNotifications)
	rg.POST("/read-all", h.MarkAllAsRead)
	rg.POST("/:id/read", h.MarkAsRead)
}

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	onlyUnread := qp.Get("unread") == "true"

	records, paginator, err := h.service.GetNotifications(ctx, userID, onlyUnread, p)
	if err != nil {
		utils.ErrorMessage(c, "Failed to fetch notifications", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"total_records": paginator.TotalRecords,
		"from":          paginator.From,
		"to":            paginator.To,
		"data":          records,
	})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Invalid ID", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.service.MarkAsRead(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Failed to mark notification as read", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Notification marked as read", "Success", http.StatusOK)
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.service.MarkAllAsRead(ctx, userID); err != nil {
		utils.ErrorMessage(c, "Failed to mark all notifications as read", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "All notifications marked as read", "Success", http.StatusOK)
}
