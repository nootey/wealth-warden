package handlers

import (
	"net/http"
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

func (h *ExportHandler) GetExportsByExportType(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	ExportType := c.Param("Export_type")

	records, err := h.Service.FetchExportsByExportType(userID, ExportType)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}
