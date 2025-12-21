package handlers

import (
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type InvestmentHandler struct {
	Service *services.InvestmentService
	v       *validators.GoValidator
}

func NewInvestmentHandler(
	service *services.InvestmentService,
	v *validators.GoValidator,
) *InvestmentHandler {
	return &InvestmentHandler{
		Service: service,
		v:       v,
	}
}

func (h *InvestmentHandler) InsertHolding(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.InvestmentHoldingReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertHolding(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}
