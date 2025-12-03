package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	Service *services.ExportService
	v       *validators.GoValidator
}

func NewExportHandler(
	service *services.ExportService,
	v *validators.GoValidator,
) *ExportHandler {
	return &ExportHandler{
		Service: service,
		v:       v,
	}
}

func (h *ExportHandler) GetExports(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchExports(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ExportHandler) GetExportsByExportType(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	exportType := c.Param("export_type")

	records, err := h.Service.FetchExportsByExportType(ctx, userID, exportType)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ExportHandler) CreateExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	_, err := h.Service.CreateExport(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Export created", "Success", http.StatusOK)
}

func (h *ExportHandler) DownloadExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("export_%s.zip", todayStr)
	data, err := h.Service.DownloadExport(ctx, id, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/zip", data)
}

func (h *ExportHandler) DeleteExport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	if err := h.Service.DeleteExport(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
