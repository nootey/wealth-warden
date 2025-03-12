package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func BudgetRoutes(apiGroup *gin.RouterGroup, handler *handlers.BudgetHandler) {
	apiGroup.GET("/current", handler.GetCurrentMonthlyBudget)
	apiGroup.POST("/create", handler.CreateNewMonthlyBudget)
}
