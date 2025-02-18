package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func inflowRoutes(apiGroup *gin.RouterGroup, handler *handlers.InflowHandler) {
	apiGroup.GET("/get-inflows-paginated", handler.GetInflowsPaginated)
	apiGroup.GET("/get-all-inflows-grouped-month", handler.GetAllInflowsGroupedByMonth)
	apiGroup.GET("/get-all-inflow-categories", handler.GetAllInflowCategories)
	apiGroup.POST("/create-new-inflow", handler.CreateNewInflow)
	apiGroup.POST("/create-new-inflow-category", handler.CreateNewInflowCategory)
	apiGroup.POST("/delete-inflow", handler.DeleteInflow)
	apiGroup.POST("/delete-inflow-category", handler.DeleteInflowCategory)
}
