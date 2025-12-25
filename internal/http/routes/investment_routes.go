package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func InvestmentRoutes(ap *gin.RouterGroup, h *handlers.InvestmentHandler) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetInvestmentAssetsPaginated)
	ap.GET("all", authz.RequireAllMW("view_data"), h.GetAllInvestmentAssets)
	ap.GET(":id", authz.RequireAllMW("view_data"), h.GetInvestmentAssetByID)
	ap.GET("trades", authz.RequireAllMW("view_data"), h.GetInvestmentTradesPaginated)
	ap.GET("trades/:id", authz.RequireAllMW("view_data"), h.GetInvestmentTradeByID)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertInvestmentAsset)
	ap.PUT("trades", authz.RequireAllMW("manage_data"), h.InsertInvestmentTrade)
	ap.PUT(":id", authz.RequireAllMW("manage_data"), h.UpdateInvestmentAsset)
	ap.PUT("trades/:id", authz.RequireAllMW("manage_data"), h.UpdateInvestmentTrade)
	ap.DELETE(":id", authz.RequireAllMW("manage_data"), h.DeleteInvestmentAsset)
	ap.DELETE("trades/:id", authz.RequireAllMW("manage_data"), h.DeleteInvestmentTrade)
}
