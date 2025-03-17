package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func BudgetRoutes(apiGroup *gin.RouterGroup, handler *handlers.BudgetHandler) {
	apiGroup.GET("/current", handler.GetCurrentMonthlyBudget)
	apiGroup.GET("/sync", handler.SynchronizeCurrentMonthlyBudget)
	apiGroup.GET("/sync-snapshot", handler.SynchronizeCurrentMonthlyBudgetSnapshot)
	apiGroup.POST("/create", handler.CreateNewMonthlyBudget)
	apiGroup.POST("/create-allocation", handler.CreateNewBudgetAllocation)
	apiGroup.POST("/update", handler.UpdateMonthlyBudget)
}
