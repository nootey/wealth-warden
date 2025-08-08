package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func TransactionRoutes(apiGroup *gin.RouterGroup, handler *handlers.TransactionHandler) {
	apiGroup.GET("", handler.GetTransactionsPaginated)
	apiGroup.GET("categories", handler.GetCategories)
	apiGroup.PUT("", handler.InsertTransaction)
}
