package v1

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
	ap.POST("categories/restore", authz.RequireAllMW("manage_data"), h.RestoreCategory)
	ap.POST("categories/restore/name", authz.RequireAllMW("manage_data"), h.RestoreCategoryName)
}
