package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func ExposedAuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.POST("/login", handler.LoginUser)
	apiGroup.POST("/logout", handler.LogoutUser)
}

func AuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/me", handler.GetAuthUser)
}
