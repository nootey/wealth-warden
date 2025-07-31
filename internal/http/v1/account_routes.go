package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func AccountRoutes(apiGroup *gin.RouterGroup, handler *handlers.AccountHandler) {
	apiGroup.GET("/types", handler.GetAccountTypes)
	apiGroup.POST("", handler.InsertAccount)
}
