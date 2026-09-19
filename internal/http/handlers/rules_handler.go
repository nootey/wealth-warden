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

type RulesHandler struct {
	service services.RulesServiceInterface
	v       validators.Validator
}

func NewRulesHandler(
	service services.RulesServiceInterface,
	v validators.Validator,
) *RulesHandler {
	return &RulesHandler{
		service: service,
		v:       v,
	}
}

func (h *RulesHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), h.GetRules)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), h.GetRuleByID)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), h.InsertRule)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_data"), h.UpdateRule)
	apiGroup.DELETE(":id", authz.RequireAllMW("manage_data"), h.DeleteRule)
	apiGroup.POST("/apply", authz.RequireAllMW("manage_data"), h.ApplyRules)
}

func (h *RulesHandler) GetRules(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.service.FetchRules(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *RulesHandler) GetRuleByID(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	record, err := h.service.FetchRuleByID(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *RulesHandler) InsertRule(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.RuleReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if _, err := h.service.InsertRule(ctx, userID, record); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Rule created", "Success", http.StatusOK)
}

func (h *RulesHandler) UpdateRule(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var record *models.RuleReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if _, err := h.service.UpdateRule(ctx, userID, id, record); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Rule updated", "Success", http.StatusOK)
}

func (h *RulesHandler) DeleteRule(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.DeleteRule(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Rule deleted", "Success", http.StatusOK)
}

func (h *RulesHandler) ApplyRules(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.service.DispatchApplyRules(ctx, userID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Applying rules to uncategorized transactions", "Success", http.StatusOK)
}
