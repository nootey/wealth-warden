package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func UserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("", handler.GetUsers)
	apiGroup.GET("/:id", handler.GetUserById)
}

func PublicUserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("/invitations/:hash", handler.GetInvitationByHash)
	apiGroup.PUT("invitations", handler.CreateInvitation)
}
