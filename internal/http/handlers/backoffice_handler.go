package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type BackofficeHandler struct {
	service *services.BackofficeService
	v       *validators.GoValidator
}

func NewBackofficeHandler(
	service *services.BackofficeService,
	v *validators.GoValidator,
) *BackofficeHandler {
	return &BackofficeHandler{
		service: service,
		v:       v,
	}
}

func (h *BackofficeHandler) Routes(ap *gin.RouterGroup) {
	ap.POST("/backfill/asset-cash-flows", authz.RequireAllMW("access_backoffice"), h.BackfillAssetCashFlows)
	ap.POST("/correct/fee-accounting", authz.RequireAllMW("access_backoffice"), h.CorrectFeeAccounting)
}

func (h *BackofficeHandler) BackfillAssetCashFlows(c *gin.Context) {
	if err := h.service.BackfillAssetCashFlows(c.Request.Context()); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"message": "backfill job queued"})
}

func (h *BackofficeHandler) CorrectFeeAccounting(c *gin.Context) {
	if err := h.service.CorrectFeeAccounting(c.Request.Context()); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"message": "fee accounting correction job queued"})
}
