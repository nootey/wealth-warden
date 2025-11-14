package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func AccountRoutes(apiGroup *gin.RouterGroup, handler *handlers.AccountHandler) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), handler.GetAccountsPaginated)
	apiGroup.GET("/all", authz.RequireAllMW("view_data"), handler.GetAllAccounts)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), handler.GetAccountByID)
	apiGroup.GET("/name/:name", authz.RequireAllMW("manage_data"), handler.GetAccountByName)
	apiGroup.GET("/subtype/:sub", authz.RequireAllMW("view_data"), handler.GetAccountsBySubtype)
	apiGroup.GET("/type/:type", authz.RequireAllMW("view_data"), handler.GetAccountsByType)
	apiGroup.GET("/types", authz.RequireAllMW("view_data"), handler.GetAccountTypes)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), handler.InsertAccount)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_data"), handler.UpdateAccount)
	apiGroup.POST(":id/active", authz.RequireAllMW("manage_data"), handler.ToggleAccountActiveState)
	apiGroup.DELETE(":id", authz.RequireAllMW("manage_data"), handler.CloseAccount)

	apiGroup.POST("/balances/backfill", authz.RequireAllMW("manage_data"), handler.BackfillBalancesForUser)
}
