package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ChartingRoutes(apiGroup *gin.RouterGroup, handler *handlers.ChartingHandler) {
	apiGroup.GET("/networth", authz.RequireAllMW("view_basic_statistics"), handler.NetWorthChart)
	apiGroup.GET("/monthly-cash-flow", authz.RequireAllMW("view_basic_statistics"), handler.GetMonthlyCashFlowForYear)
}
