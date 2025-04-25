package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func OutflowRoutes(apiGroup *gin.RouterGroup, handler *handlers.OutflowHandler) {
	apiGroup.GET("/", handler.GetOutflowsPaginated)
	apiGroup.GET("/grouped-by-month", handler.GetAllOutflowsGroupedByMonth)
	apiGroup.GET("/categories", handler.GetAllOutflowCategories)
	apiGroup.POST("/create", handler.CreateNewOutflow)
	apiGroup.POST("/update", handler.UpdateOutflow)
	apiGroup.POST("/create-reoccurring", handler.CreateNewReoccurringOutflow)
	apiGroup.POST("/create-category", handler.CreateNewOutflowCategory)
	apiGroup.POST("/update-category", handler.UpdateOutflowCategory)
	apiGroup.POST("/delete", handler.DeleteOutflow)
	apiGroup.POST("/delete-category", handler.DeleteOutflowCategory)
}
