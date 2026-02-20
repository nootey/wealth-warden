package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type RolePermissionHandler struct {
	Service *services.RolePermissionService
	v       *validators.GoValidator
}

func NewRolePermissionHandler(
	service *services.RolePermissionService,
	v *validators.GoValidator,
) *RolePermissionHandler {
	return &RolePermissionHandler{
		Service: service,
		v:       v,
	}
}

func (h *RolePermissionHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("manage_roles"), h.GetAllRoles)
	apiGroup.GET("/permissions", authz.RequireAllMW("manage_roles"), h.GetAllPermissions)
	apiGroup.GET(":id", authz.RequireAllMW("manage_roles"), h.GetRoleById)
	apiGroup.PUT("", authz.RequireAllMW("manage_roles"), h.InsertRole)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_roles"), h.UpdateRole)
	apiGroup.DELETE(":id", authz.RequireAllMW("delete_roles"), h.DeleteRole)
}

func (h *RolePermissionHandler) GetAllRoles(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	withPermissions := strings.EqualFold(qp.Get("with_permissions"), "true")

	records, err := h.Service.FetchAllRoles(ctx, withPermissions)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *RolePermissionHandler) GetAllPermissions(c *gin.Context) {

	ctx := c.Request.Context()
	records, err := h.Service.FetchAllPermissions(ctx)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *RolePermissionHandler) GetRoleById(c *gin.Context) {

	ctx := c.Request.Context()
	idStr := c.Param("id")
	qp := c.Request.URL.Query()
	wp := strings.EqualFold(qp.Get("with_permissions"), "true")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "param error", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	user, err := h.Service.FetchRoleByID(ctx, id, wp)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *RolePermissionHandler) InsertRole(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.RoleReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := utils.SanitizeStruct(&req); err != nil {
		utils.ErrorMessage(c, "Sanitization error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	_, err := h.Service.InsertRole(ctx, userID, req)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Create success", "Record has been created successfully.", http.StatusOK)
}

func (h *RolePermissionHandler) UpdateRole(c *gin.Context) {

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

	var record *models.RoleReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateRole(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *RolePermissionHandler) DeleteRole(c *gin.Context) {

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

	if err := h.Service.DeleteRole(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
