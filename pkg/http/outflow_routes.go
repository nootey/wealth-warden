package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func outflowRoutes(apiGroup *gin.RouterGroup, handler *handlers.OutflowHandler) {
	apiGroup.GET("/get-outflows-paginated", handler.GetOutflowsPaginated)
	apiGroup.GET("/get-all-outflows-grouped-month", handler.GetAllOutflowsGroupedByMonth)
	apiGroup.GET("/get-all-outflow-categories", handler.GetAllOutflowCategories)
	apiGroup.POST("/create-new-outflow", handler.CreateNewOutflow)
	apiGroup.POST("/update-outflow", handler.UpdateOutflow)
	apiGroup.POST("/create-new-reoccurring-outflow", handler.CreateNewReoccurringOutflow)
	apiGroup.POST("/create-new-outflow-category", handler.CreateNewOutflowCategory)
	apiGroup.POST("/update-outflow-category", handler.UpdateOutflowCategory)
	apiGroup.POST("/delete-outflow", handler.DeleteOutflow)
	apiGroup.POST("/delete-outflow-category", handler.DeleteOutflowCategory)
}
