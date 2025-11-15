package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func TransactionRoutes(ap *gin.RouterGroup, h *handlers.TransactionHandler) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetTransactionsPaginated)
	ap.GET(":id", authz.RequireAllMW("view_data"), h.GetTransactionByID)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertTransaction)
	ap.PUT("/:id", authz.RequireAllMW("manage_data"), h.UpdateTransaction)
	ap.DELETE("/:id", authz.RequireAllMW("manage_data"), h.DeleteTransaction)
	ap.GET("transfers", authz.RequireAllMW("view_data"), h.GetTransfersPaginated)
	ap.PUT("transfers", authz.RequireAllMW("manage_data"), h.InsertTransfer)
	ap.DELETE("transfers/:id", authz.RequireAllMW("manage_data"), h.DeleteTransfer)
	ap.POST("/restore", authz.RequireAllMW("manage_data"), h.RestoreTransaction)
	ap.GET("categories", authz.RequireAllMW("view_data"), h.GetCategories)
	ap.GET("categories/:id", authz.RequireAllMW("view_data"), h.GetCategoryByID)
	ap.PUT("categories", authz.RequireAllMW("manage_data"), h.InsertCategory)
	ap.PUT("categories/:id", authz.RequireAllMW("manage_data"), h.UpdateCategory)
	ap.DELETE("categories/:id", authz.RequireAllMW("manage_data"), h.DeleteCategory)
	ap.GET("categories/groups", authz.RequireAllMW("view_data"), h.GetCategoryGroups)
	ap.GET("categories/groups/:id", authz.RequireAllMW("view_data"), h.GetCategoryGroupByID)
	ap.PUT("categories/groups", authz.RequireAllMW("manage_data"), h.InsertCategoryGroup)
	ap.PUT("categories/groups/:id", authz.RequireAllMW("manage_data"), h.UpdateCategoryGroup)
	ap.DELETE("categories/groups/:id", authz.RequireAllMW("manage_data"), h.DeleteCategoryGroup)
	ap.POST("categories/restore", authz.RequireAllMW("manage_data"), h.RestoreCategory)
	ap.POST("categories/restore/name", authz.RequireAllMW("manage_data"), h.RestoreCategoryName)
	ap.GET("templates", authz.RequireAllMW("view_data"), h.GetTransactionTemplatesPaginated)
	ap.GET("templates/:id", authz.RequireAllMW("view_data"), h.GetTransactionTemplateByID)
	ap.GET("templates/count", authz.RequireAllMW("view_data"), h.GetTransactionTemplateCount)
	ap.PUT("templates", authz.RequireAllMW("manage_data"), h.InsertTransactionTemplate)
	ap.PUT("templates/:id", authz.RequireAllMW("manage_data"), h.UpdateTransactionTemplate)
	ap.POST("templates/:id/active", authz.RequireAllMW("manage_data"), h.ToggleTransactionTemplateActiveState)
	ap.DELETE("templates/:id", authz.RequireAllMW("manage_data"), h.DeleteTransactionTemplate)
}
