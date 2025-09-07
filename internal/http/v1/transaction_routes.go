package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func TransactionRoutes(apiGroup *gin.RouterGroup, handler *handlers.TransactionHandler) {
	apiGroup.GET("", handler.GetTransactionsPaginated)
	apiGroup.GET(":id", handler.GetTransactionByID)
	apiGroup.GET("categories", handler.GetCategories)
	apiGroup.GET("categories/:id", handler.GetCategoryByID)
	apiGroup.PUT("", handler.InsertTransaction)
	apiGroup.PUT(":id", handler.UpdateTransaction)
	apiGroup.DELETE(":id", handler.DeleteTransaction)
	apiGroup.GET("transfers", handler.GetTransfersPaginated)
	apiGroup.PUT("transfers", handler.InsertTransfer)
	apiGroup.DELETE("transfers/:id", handler.DeleteTransfer)
	apiGroup.POST("/restore", handler.RestoreTransaction)
}
