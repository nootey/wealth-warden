package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func AccountRoutes(apiGroup *gin.RouterGroup, handler *handlers.AccountHandler) {
	apiGroup.GET("", handler.GetAccountsPaginated)
	apiGroup.GET("/all", handler.GetAllAccounts)
	apiGroup.GET("/:id", handler.GetAccountByID)
	apiGroup.GET("/types", handler.GetAccountTypes)
	apiGroup.PUT("", handler.InsertAccount)
	apiGroup.PUT(":id", handler.UpdateAccount)
	apiGroup.POST(":id/active", handler.ToggleAccountActiveState)
	apiGroup.DELETE(":id", handler.CloseAccount)

	apiGroup.POST("/balances/backfill", handler.BackfillBalancesForUser)
}
