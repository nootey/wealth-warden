package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func StatsRoutes(apiGroup *gin.RouterGroup, handler *handlers.StatisticsHandler) {
	apiGroup.GET("/account", authz.RequireAllMW("view_basic_statistics"), handler.GetAccountBasicStatistics)
	apiGroup.GET("/years", authz.RequireAllMW("view_basic_statistics"), handler.GetAvailableStatsYears)
}
