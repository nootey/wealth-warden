package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func SavingsRoutes(apiGroup *gin.RouterGroup, handler *handlers.SavingsHandler) {
	apiGroup.GET("/", handler.GetSavingsPaginated)
	apiGroup.GET("/categories", handler.GetAllSavingsCategories)
	apiGroup.GET("/create-category", handler.CreateNewSavingsCategory)

}
