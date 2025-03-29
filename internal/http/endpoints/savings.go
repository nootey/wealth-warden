package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func SavingsRoutes(apiGroup *gin.RouterGroup, handler *handlers.SavingsHandler) {
	apiGroup.GET("/", handler.GetSavingsPaginated)
	apiGroup.GET("/grouped-by-month", handler.GetAllSavingsGroupedByMonth)
	apiGroup.GET("/categories", handler.GetAllSavingsCategories)
	apiGroup.POST("/create-category", handler.CreateNewSavingsCategory)
	apiGroup.POST("/create", handler.CreateNewSavingsAllocation)
	apiGroup.POST("/update-category", handler.UpdateSavingsCategory)

}
