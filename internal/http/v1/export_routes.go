package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ExportRoutes(apiGroup *gin.RouterGroup, handler *handlers.ExportHandler) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), handler.GetExports)
	apiGroup.PUT("", authz.RequireAllMW("create_exports"), handler.CreateExport)
}
