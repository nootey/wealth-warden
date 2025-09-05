package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
)

type SettingsHandler struct {
	Service *services.SettingsService
}

func NewSettingsHandler(SettingsService *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		Service: SettingsService,
	}
}

func (h *SettingsHandler) GetGeneralSettings(c *gin.Context) {
	record, err := h.Service.FetchGeneralSettings(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetUserSettings(c *gin.Context) {
	record, err := h.Service.FetchUserSettings(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) UpdateUserSettings(c *gin.Context) {

	var record models.SettingsUserReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdateUserSettings(c, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}
