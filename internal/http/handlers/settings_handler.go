package handlers

import (
	"errors"
	"fmt"
	"net/http"
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
	apiGroup.PUT("/users/preferences", authz.RequireAllMW("manage_data"), h.UpdatePreferenceSettings)
	apiGroup.PUT("/users/profile", authz.RequireAllMW("manage_data"), h.UpdateProfileSettings)
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

func (h *SettingsHandler) GetDatabaseBackups(c *gin.Context) {
	ctx := c.Request.Context()

	backups, err := h.Service.GetDatabaseBackups(ctx)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
	})
}

func (h *SettingsHandler) CreateDatabaseBackup(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.Service.CreateDatabaseBackup(ctx, userID); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Backup dump created", "Success", http.StatusOK)

}

func (h *SettingsHandler) RestoreDatabaseBackup(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req struct {
		BackupName string `json:"backup_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreDatabaseBackup(ctx, userID, req.BackupName); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Backup dump created", "Success", http.StatusOK)

}

func (h *SettingsHandler) DownloadDatabaseBackup(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req struct {
		BackupName string `json:"backup_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Error occurred", "invalid request body", http.StatusBadRequest, err)
		return
	}

	if req.BackupName == "" {
		err := errors.New("backup_name is required")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	filename := fmt.Sprintf("%s.zip", req.BackupName)
	data, err := h.Service.DownloadBackup(ctx, req.BackupName, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/zip", data)
}
