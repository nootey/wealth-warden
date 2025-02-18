package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func exposedAuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.POST("/login", handler.LoginUser)
	apiGroup.POST("/logout", handler.LogoutUser)
}

func authRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/get-auth-user", handler.GetAuthUser)
}
