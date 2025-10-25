package v1

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ExportRoutes(apiGroup *gin.RouterGroup, handler *handlers.ExportHandler) {
	apiGroup.GET("/:export_type", authz.RequireAllMW("view_data"), handler.GetExportsByExportType)
}
