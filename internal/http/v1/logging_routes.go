package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func LoggingRoutes(apiGroup *gin.RouterGroup, handler *handlers.LoggingHandler) {
	apiGroup.GET("", handler.GetActivityLogs)
	apiGroup.GET("/filter-data", handler.GetActivityLogFilterData)
	apiGroup.DELETE("/:id", handler.DeleteActivityLog)
}
