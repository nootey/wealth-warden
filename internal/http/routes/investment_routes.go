package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func InvestmentRoutes(ap *gin.RouterGroup, h *handlers.InvestmentHandler) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetInvestmentHoldingsPaginated)
	ap.GET("transactions", authz.RequireAllMW("view_data"), h.GetInvestmentTransactionsPaginated)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertInvestmentHolding)
}
