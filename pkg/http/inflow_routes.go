package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func inflowRoutes(apiGroup *gin.RouterGroup, handler *handlers.InflowHandler) {
	apiGroup.GET("/get-inflows-paginated", handler.GetInflowsPaginated)
	apiGroup.GET("/get-all-inflows-grouped-month", handler.GetAllInflowsGroupedByMonth)
	apiGroup.GET("/get-all-inflow-categories", handler.GetAllInflowCategories)
	apiGroup.GET("/get-all-dynamic-categories", handler.GetAllDynamicCategories)
	apiGroup.POST("/create-new-inflow", handler.CreateNewInflow)
	apiGroup.POST("/update-inflow", handler.UpdateInflow)
	apiGroup.POST("/create-new-reoccurring-inflow", handler.CreateNewReoccurringInflow)
	apiGroup.POST("/create-new-inflow-category", handler.CreateNewInflowCategory)
	apiGroup.POST("/create-new-dynamic-category", handler.CreateNewDynamicCategory)
	apiGroup.POST("/update-inflow-category", handler.UpdateInflowCategory)
	apiGroup.POST("/delete-inflow", handler.DeleteInflow)
	apiGroup.POST("/delete-inflow-category", handler.DeleteInflowCategory)
	apiGroup.POST("/delete-dynamic-category", handler.DeleteDynamicCategory)
}
