package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func InflowRoutes(apiGroup *gin.RouterGroup, handler *handlers.InflowHandler) {
	apiGroup.GET("/", handler.GetInflowsPaginated)
	apiGroup.GET("/grouped-by-month", handler.GetAllInflowsGroupedByMonth)
	apiGroup.GET("/categories", handler.GetAllInflowCategories)
	apiGroup.GET("/dynamic-categories", handler.GetAllDynamicCategories)
	apiGroup.POST("/create", handler.CreateNewInflow)
	apiGroup.POST("/update", handler.UpdateInflow)
	apiGroup.POST("/create-reoccurring", handler.CreateNewReoccurringInflow)
	apiGroup.POST("/create-category", handler.CreateNewInflowCategory)
	apiGroup.POST("/create-dynamic-category", handler.CreateNewDynamicCategory)
	apiGroup.POST("/update-category", handler.UpdateInflowCategory)
	apiGroup.POST("/delete", handler.DeleteInflow)
	apiGroup.POST("/delete-category", handler.DeleteInflowCategory)
	apiGroup.POST("/delete-dynamic-category", handler.DeleteDynamicCategory)
}
