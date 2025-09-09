package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func ChartingRoutes(apiGroup *gin.RouterGroup, handler *handlers.ChartingHandler) {
	apiGroup.GET("/networth", handler.NetWorthChart)
}
