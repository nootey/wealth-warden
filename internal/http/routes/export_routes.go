package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func ExportRoutes(apiGroup *gin.RouterGroup, handler *handlers.ExportHandler) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), handler.GetExports)
	apiGroup.POST("", authz.RequireAllMW("create_exports"), handler.CreateExport)
	apiGroup.POST(":id/download", authz.RequireAllMW("manage_data"), handler.DownloadExport)
	apiGroup.DELETE(":id", authz.RequireAllMW("delete_exports"), handler.DeleteExport)
}
