package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func UserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler, roleHandler *handlers.RolePermissionHandler) {
	apiGroup.GET("", handler.GetUsersPaginated)
	apiGroup.GET("/:id", handler.GetUserById)
	apiGroup.PUT(":id", handler.UpdateUser)
	apiGroup.DELETE(":id", handler.DeleteUser)

	apiGroup.GET("invitations", handler.GetInvitationsPaginated)
	apiGroup.PUT("invitations", handler.InsertInvitation)
	apiGroup.POST("invitations/resend/:id", handler.ResendInvitation)
	apiGroup.DELETE("invitations/:id", handler.DeleteInvitation)

	apiGroup.GET("roles", roleHandler.GetAllRoles)
	apiGroup.GET("permissions", roleHandler.GetAllPermissions)
	apiGroup.GET("roles/:id", roleHandler.GetRoleById)
	apiGroup.PUT("roles", roleHandler.InsertRole)
	apiGroup.PUT("roles/:id", roleHandler.UpdateRole)
	apiGroup.DELETE("roles/:id", roleHandler.DeleteRole)
}

func PublicUserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("/invitations/:hash", handler.GetInvitationByHash)
	apiGroup.GET("/token", handler.GetUserByToken)
}
