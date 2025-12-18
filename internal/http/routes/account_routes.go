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
	apiGroup.POST(":id/projection/save", authz.RequireAllMW("manage_data"), handler.SaveAccountProjection)
	apiGroup.POST(":id/projection/revert", authz.RequireAllMW("manage_data"), handler.RevertAccountProjection)
	apiGroup.GET("/balances/:id/latest", authz.RequireAllMW("view_data"), handler.GetLatestBalance)
	apiGroup.POST("/balances/backfill", authz.RequireAllMW("manage_data"), handler.BackfillBalancesForUser)
	apiGroup.GET("/defaults/all", authz.RequireAllMW("view_data"), handler.GetAccountsWithDefaults)
	apiGroup.GET("/defaults/types", authz.RequireAllMW("view_data"), handler.GetAccountTypesWithoutDefaults)
	apiGroup.PATCH("/defaults/set/:id", authz.RequireAllMW("view_data"), handler.SetDefaultAccount)
	apiGroup.PATCH("/defaults/unset/:id", authz.RequireAllMW("view_data"), handler.UnsetDefaultAccount)
}
