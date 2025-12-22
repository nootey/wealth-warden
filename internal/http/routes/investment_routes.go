package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func InvestmentRoutes(ap *gin.RouterGroup, h *handlers.InvestmentHandler) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetInvestmentHoldingsPaginated)
	ap.GET("all", authz.RequireAllMW("view_data"), h.GetAllInvestmentHoldings)
	ap.GET(":id", authz.RequireAllMW("view_data"), h.GetInvestmentHoldingByID)
	ap.GET("transactions", authz.RequireAllMW("view_data"), h.GetInvestmentTransactionsPaginated)
	ap.GET("transactions/:id", authz.RequireAllMW("view_data"), h.GetInvestmentTransactionByID)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertInvestmentHolding)
	ap.PUT("transactions", authz.RequireAllMW("manage_data"), h.InsertInvestmentTransaction)
}
