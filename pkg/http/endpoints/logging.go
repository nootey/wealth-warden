package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func LoggingRoutes(apiGroup *gin.RouterGroup, handler *handlers.LoggingHandler) {
	apiGroup.GET("/get-activity-logs", handler.GetActivityLogs)
	apiGroup.GET("/get-access-logs", handler.GetAccessLogs)
	apiGroup.GET("/get-notification-logs", handler.GetNotificationLogs)
}
