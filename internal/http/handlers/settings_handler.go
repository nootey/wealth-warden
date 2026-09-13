package handlers

import (
	"net/http"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
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

func (h *SettingsHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("root_access"), h.GetGeneralSettings)
	apiGroup.GET("/users", authz.RequireAllMW("view_data"), h.GetUserSettings)
	apiGroup.GET("/timezones", authz.RequireAllMW("view_data"), h.GetAvailableTimezones)
	apiGroup.GET("/currencies", authz.RequireAllMW("view_data"), h.GetAvailableCurrencies)
	apiGroup.PUT("/users/preferences", authz.RequireAllMW("manage_data"), h.UpdatePreferenceSettings)
	apiGroup.PUT("/users/profile", authz.RequireAllMW(), h.UpdateProfileSettings)
}

func (h *SettingsHandler) GetGeneralSettings(c *gin.Context) {

	ctx := c.Request.Context()
	record, err := h.Service.FetchGeneralSettings(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetUserSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.FetchUserSettings(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SettingsHandler) GetAvailableTimezones(c *gin.Context) {

	ctx := c.Request.Context()
	tzones, err := h.Service.FetchAvailableTimezones(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tzones)
}

func (h *SettingsHandler) GetAvailableCurrencies(c *gin.Context) {

	ctx := c.Request.Context()
	currencies, err := h.Service.FetchAvailableCurrencies(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, currencies)
}

func (h *SettingsHandler) UpdatePreferenceSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record models.PreferenceSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.UpdatePreferenceSettings(ctx, userID, record); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}

func (h *SettingsHandler) UpdateProfileSettings(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record models.ProfileSettingsReq
	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.UpdateProfileSettings(ctx, userID, record); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}
