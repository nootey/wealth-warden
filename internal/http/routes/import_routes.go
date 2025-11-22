package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ImportRoutes(apiGroup *gin.RouterGroup, handler *handlers.ImportHandler) {
	apiGroup.GET("/:import_type", authz.RequireAllMW("view_data"), handler.GetImportsByImportType)
	apiGroup.GET("/:import_type/:id", authz.RequireAllMW("view_data"), handler.GetStoredCustomImport)
	apiGroup.POST("custom/validate", authz.RequireAllMW("manage_data"), handler.ValidateCustomImport)
	apiGroup.POST("custom/accounts", authz.RequireAllMW("manage_data"), handler.ImportAccounts)
	apiGroup.POST("custom/categories", authz.RequireAllMW("manage_data"), handler.ImportCategories)
	apiGroup.POST("custom/transactions", authz.RequireAllMW("manage_data"), handler.ImportTransactions)
	apiGroup.POST("custom/investments", authz.RequireAllMW("manage_data"), handler.TransferInvestmentsFromImport)
	apiGroup.POST("custom/savings", authz.RequireAllMW("manage_data"), handler.TransferSavingsFromImport)
	apiGroup.DELETE("/:id", authz.RequireAllMW("manage_data"), handler.DeleteImport)
}
