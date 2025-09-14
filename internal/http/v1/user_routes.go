package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func UserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("", handler.GetUsersPaginated)
	apiGroup.GET("roles", handler.GetRoles)
	apiGroup.GET("/:id", handler.GetUserById)
	apiGroup.PUT("invitations", handler.InsertInvitation)
	apiGroup.PUT(":id", handler.UpdateUser)
	apiGroup.DELETE(":id", handler.DeleteUser)
}

func PublicUserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("/invitations/:hash", handler.GetInvitationByHash)
	apiGroup.GET("/token", handler.GetUserByToken)
}
