package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"
)

func TransactionRoutes(ap *gin.RouterGroup, h *handlers.TransactionHandler) {
	ap.GET("", h.GetTransactionsPaginated)
	ap.GET(":id", h.GetTransactionByID)
	ap.POST("",
		authz.RequireAllMW("manage_data"),
		h.InsertTransaction,
	)
	ap.PUT("/:id",
		authz.RequireAllMW("manage_data"),
		h.UpdateTransaction,
	)
	ap.DELETE("/:id",
		authz.RequireAllMW("manage_data"),
		h.DeleteTransaction,
	)
	ap.GET("transfers", h.GetTransfersPaginated)
	ap.PUT("transfers", h.InsertTransfer)
	ap.DELETE("transfers/:id", h.DeleteTransfer)
	ap.POST("/restore", h.RestoreTransaction)
	ap.GET("categories", h.GetCategories)
	ap.GET("categories/:id", h.GetCategoryByID)
	ap.PUT("categories", h.InsertCategory)
	ap.PUT("categories/:id", h.UpdateCategory)
	ap.DELETE("categories/:id", h.DeleteCategory)
	ap.POST("categories/restore", h.RestoreCategory)
	ap.POST("categories/restore/name", h.RestoreCategoryName)
}
