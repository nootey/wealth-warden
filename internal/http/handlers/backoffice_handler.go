package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type BackofficeHandler struct {
	service services.BackofficeServiceInterface
	v       validators.Validator
}

func NewBackofficeHandler(
	service services.BackofficeServiceInterface,
	v validators.Validator,
) *BackofficeHandler {
	return &BackofficeHandler{
		service: service,
		v:       v,
	}
}

func (h *BackofficeHandler) Routes(ap *gin.RouterGroup) {
	ap.POST("/backfill/asset-cash-flows", authz.RequireAllMW("access_backoffice"), h.BackfillAssetCashFlows)
	ap.POST("/correct/fee-accounting", authz.RequireAllMW("access_backoffice"), h.CorrectFeeAccounting)
	ap.POST("/migrate/zero-cost-trades", authz.RequireAllMW("access_backoffice"), h.MigrateZeroCostTrades)
	ap.POST("/backfill/income-fx-rates", authz.RequireAllMW("access_backoffice"), h.BackfillIncomeExchangeRates)
}

func (h *BackofficeHandler) BackfillAssetCashFlows(c *gin.Context) {
	if err := h.service.BackfillAssetCashFlows(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(202, gin.H{"message": "backfill job queued"})
}

func (h *BackofficeHandler) CorrectFeeAccounting(c *gin.Context) {
	if err := h.service.CorrectFeeAccounting(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(202, gin.H{"message": "fee accounting correction job queued"})
}

func (h *BackofficeHandler) BackfillIncomeExchangeRates(c *gin.Context) {
	if err := h.service.BackfillIncomeExchangeRates(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(202, gin.H{"message": "income exchange rate backfill job queued"})
}

func (h *BackofficeHandler) MigrateZeroCostTrades(c *gin.Context) {
	if err := h.service.MigrateZeroCostTrades(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(202, gin.H{"message": "zero-cost trade migration queued"})
}
