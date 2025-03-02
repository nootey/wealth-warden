package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func userRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("/get-users", handler.GetUsers)
}
