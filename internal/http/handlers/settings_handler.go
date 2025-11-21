package handlers

import (
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	Service *services.SettingsService
	v       *validators.GoValidator
}

func NewSettingsHandler(
	service *services.SettingsService,
	v *validators.GoValidator,
) *SettingsHandler {
	return &SettingsHandler{
		Service: service,
		v:       v,
	}
}

func (h *SettingsHandler) GetGeneralSettings(c *gin.Context) {
	record, err := h.Service.FetchGeneralSettings()
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetUserSettings(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	record, err := h.Service.FetchUserSettings(userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetAvailableTimezones(c *gin.Context) {
	tzones, err := h.Service.FetchAvailableTimezones()
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, tzones)
}

func (h *SettingsHandler) UpdatePreferenceSettings(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var record models.PreferenceSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdatePreferenceSettings(userID, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}

func (h *SettingsHandler) UpdateProfileSettings(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var record models.ProfileSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdateProfileSettings(userID, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}
