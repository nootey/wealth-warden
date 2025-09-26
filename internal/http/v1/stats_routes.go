package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func StatsRoutes(apiGroup *gin.RouterGroup, handler *handlers.StatisticsHandler) {
	apiGroup.GET("/account/:id", authz.RequireAllMW("view_basic_statistics"), handler.GetAccountBasicStatistics)
}
