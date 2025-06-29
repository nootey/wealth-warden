package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func SavingsRoutes(apiGroup *gin.RouterGroup, handler *handlers.SavingsHandler) {
	apiGroup.GET("", handler.GetSavingsPaginated)
	apiGroup.GET("/grouped-by-month", handler.GetAllSavingsGroupedByMonth)
	apiGroup.GET("/categories", handler.GetAllSavingsCategories)
	apiGroup.POST("/create-allocation", handler.CreateNewSavingsAllocation)
	apiGroup.POST("/create-deduction", handler.CreateNewSavingsDeduction)
	apiGroup.POST("/create-category", handler.CreateNewSavingsCategory)
	apiGroup.POST("/update-category", handler.UpdateSavingsCategory)
	apiGroup.POST("/delete-category", handler.DeleteSavingsCategory)

}
