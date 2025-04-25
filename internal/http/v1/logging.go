package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func LoggingRoutes(apiGroup *gin.RouterGroup, handler *handlers.LoggingHandler) {
	apiGroup.GET("/activity", handler.GetActivityLogs)
	apiGroup.GET("/access", handler.GetAccessLogs)
	apiGroup.GET("/notification", handler.GetNotificationLogs)
}
