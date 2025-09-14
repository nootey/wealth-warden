package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func RoleRoutes(apiGroup *gin.RouterGroup, handler *handlers.RolePermissionHandler) {
	apiGroup.GET("", handler.GetAllRoles)
	apiGroup.GET("permissions", handler.GetAllPermissions)
}
