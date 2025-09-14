package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
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

func (h *RolePermissionHandler) GetAllRoles(c *gin.Context) {

	qp := c.Request.URL.Query()
	withPermissions := strings.EqualFold(qp.Get("with_permissions"), "true")

	records, err := h.Service.FetchAllRoles(withPermissions)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *RolePermissionHandler) GetAllPermissions(c *gin.Context) {
	records, err := h.Service.FetchAllPermissions()
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)
}
