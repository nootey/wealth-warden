package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ImportRoutes(apiGroup *gin.RouterGroup, handler *handlers.ImportHandler) {
	apiGroup.POST("custom/validate", authz.RequireAllMW("manage_data"), handler.ValidateCustomImport)
	apiGroup.POST("custom/json", authz.RequireAllMW("manage_data"), handler.ImportFromJSON)
}
