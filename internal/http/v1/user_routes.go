package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func UserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler, roleHandler *handlers.RolePermissionHandler) {
	apiGroup.GET("", authz.RequireAllMW("manage_users"), handler.GetUsersPaginated)
	apiGroup.GET("/:id", authz.RequireAllMW("manage_users"), handler.GetUserById)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_users"), handler.UpdateUser)
	apiGroup.DELETE(":id", authz.RequireAllMW("delete_users"), handler.DeleteUser)

	apiGroup.GET("invitations", handler.GetInvitationsPaginated)
	apiGroup.PUT("invitations", handler.InsertInvitation)
	apiGroup.POST("invitations/resend/:id", handler.ResendInvitation)
	apiGroup.DELETE("invitations/:id", authz.RequireAllMW("delete_users"), handler.DeleteInvitation)

	apiGroup.GET("roles", authz.RequireAllMW("manage_roles"), roleHandler.GetAllRoles)
	apiGroup.GET("permissions", authz.RequireAllMW("manage_roles"), roleHandler.GetAllPermissions)
	apiGroup.GET("roles/:id", authz.RequireAllMW("manage_roles"), roleHandler.GetRoleById)
	apiGroup.PUT("roles", authz.RequireAllMW("manage_roles"), roleHandler.InsertRole)
	apiGroup.PUT("roles/:id", authz.RequireAllMW("manage_roles"), roleHandler.UpdateRole)
	apiGroup.DELETE("roles/:id", authz.RequireAllMW("delete_roles"), roleHandler.DeleteRole)
}

func PublicUserRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {
	apiGroup.GET("/invitations/:hash", handler.GetInvitationByHash)
	apiGroup.GET("/token", handler.GetUserByToken)
}
