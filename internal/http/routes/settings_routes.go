package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func SettingsRoutes(apiGroup *gin.RouterGroup, handler *handlers.SettingsHandler) {
	apiGroup.GET("", authz.RequireAllMW("root_access"), handler.GetGeneralSettings)
	apiGroup.GET("/users", authz.RequireAllMW("view_data"), handler.GetUserSettings)
	apiGroup.GET("/timezones", authz.RequireAllMW("view_data"), handler.GetAvailableTimezones)
	apiGroup.PUT("/users/preferences", authz.RequireAllMW("manage_data"), handler.UpdatePreferenceSettings)
	apiGroup.PUT("/users/profile", authz.RequireAllMW("manage_data"), handler.UpdateProfileSettings)
}
