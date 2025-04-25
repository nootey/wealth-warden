package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func PublicAuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.POST("/login", handler.LoginUser)
	apiGroup.POST("/logout", handler.LogoutUser)
}

func AuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/current", handler.GetAuthUser)
}
