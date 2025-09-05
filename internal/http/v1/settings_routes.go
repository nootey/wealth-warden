package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func SettingsRoutes(apiGroup *gin.RouterGroup, handler *handlers.SettingsHandler) {
	apiGroup.GET("", handler.GetGeneralSettings)
	apiGroup.GET("/users", handler.GetGeneralSettings)
	apiGroup.PUT("/users/:id", handler.UpdateUserSettings)
}
