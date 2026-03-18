package handlers

import (
	"wealth-warden/internal/services"
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
	ap.POST("/backfill/asset-cash-flows", h.BackfillAssetCashFlows)
}

func (h *BackofficeHandler) BackfillAssetCashFlows(c *gin.Context) {
	if err := h.service.BackfillAssetCashFlows(c.Request.Context()); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"message": "backfill job queued"})
}
