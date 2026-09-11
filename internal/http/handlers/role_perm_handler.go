package handlers

import (
	"net/http"
	"strings"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type RolePermissionHandler struct {
	service services.RolePermissionServiceInterface
	v       validators.Validator
}

func NewRolePermissionHandler(
	service services.RolePermissionServiceInterface,
	v validators.Validator,
) *RolePermissionHandler {
	return &RolePermissionHandler{
		service: service,
		v:       v,
	}
}

func (h *RolePermissionHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAnyMW("manage_users", "manage_roles"), h.GetAllRoles)
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

	records, err := h.service.FetchAllRoles(ctx, withPermissions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *RolePermissionHandler) GetAllPermissions(c *gin.Context) {

	ctx := c.Request.Context()
	records, err := h.service.FetchAllPermissions(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *RolePermissionHandler) GetRoleById(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	wp := strings.EqualFold(qp.Get("with_permissions"), "true")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	record, err := h.service.FetchRoleByID(ctx, id, wp)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *RolePermissionHandler) InsertRole(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.RoleReq

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := utils.SanitizeStruct(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Internal, apperr.GenericMessage, err))
		return
	}

	_, err := h.service.InsertRole(ctx, userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Create success", "Record has been created successfully.", http.StatusOK)
}

func (h *RolePermissionHandler) UpdateRole(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var record *models.RoleReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.service.UpdateRole(ctx, userID, id, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *RolePermissionHandler) DeleteRole(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.DeleteRole(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
