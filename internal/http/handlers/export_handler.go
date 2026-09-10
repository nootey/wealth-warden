package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	Service services.ExportServiceInterface
	v       validators.Validator
}

func NewExportHandler(
	service services.ExportServiceInterface,
	v validators.Validator,
) *ExportHandler {
	return &ExportHandler{
		Service: service,
		v:       v,
	}
}

func (h *ExportHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), h.GetExports)
	apiGroup.POST("", authz.RequireAllMW("create_exports"), h.CreateExport)
	apiGroup.POST(":id/download", authz.RequireAllMW("view_data"), h.DownloadExport)
	apiGroup.DELETE(":id", authz.RequireAllMW("delete_exports"), h.DeleteExport)
}

func (h *ExportHandler) GetExports(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchExports(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ExportHandler) CreateExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	_, err := h.Service.CreateExport(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Export created", "Success", http.StatusOK)
}

func (h *ExportHandler) DownloadExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("export_%s.zip", todayStr)
	data, err := h.Service.DownloadExport(ctx, id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/zip", data)
}

func (h *ExportHandler) DeleteExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	if err := h.Service.DeleteExport(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
