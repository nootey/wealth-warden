package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func LoggingRoutes(apiGroup *gin.RouterGroup, handler *handlers.LoggingHandler) {
	apiGroup.GET("", authz.RequireAllMW("view_activity_logs"), handler.GetActivityLogs)
	apiGroup.GET("/filter-data", authz.RequireAllMW("view_activity_logs"), handler.GetActivityLogFilterData)
	apiGroup.DELETE("/:id", authz.RequireAllMW("delete_activity_logs"), handler.DeleteActivityLog)
}
