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

	ctx := c.Request.Context()
	record, err := h.Service.FetchGeneralSettings(ctx)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetUserSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.FetchUserSettings(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetAvailableTimezones(c *gin.Context) {

	ctx := c.Request.Context()
	tzones, err := h.Service.FetchAvailableTimezones(ctx)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, tzones)
}

func (h *SettingsHandler) UpdatePreferenceSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record models.PreferenceSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdatePreferenceSettings(ctx, userID, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}

func (h *SettingsHandler) UpdateProfileSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record models.ProfileSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdateProfileSettings(ctx, userID, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}
