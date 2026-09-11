package handlers

import (
	"net/http"
	"strconv"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type InvestmentHandler struct {
	Service services.InvestmentServiceInterface
	v       validators.Validator
}

func NewInvestmentHandler(
	service services.InvestmentServiceInterface,
	v validators.Validator,
) *InvestmentHandler {
	return &InvestmentHandler{
		Service: service,
		v:       v,
	}
}

func (h *InvestmentHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetInvestmentAssetsPaginated)
	ap.GET("all", authz.RequireAllMW("view_data"), h.GetAllInvestmentAssets)
	ap.GET("allocation", authz.RequireAllMW("view_data"), h.GetPortfolioAllocation)
	ap.GET("returns", authz.RequireAllMW("view_data"), h.GetPortfolioReturns)
	ap.GET(":id", authz.RequireAllMW("view_data"), h.GetInvestmentAssetByID)
	ap.GET("trades", authz.RequireAllMW("view_data"), h.GetInvestmentTradesPaginated)
	ap.GET("trades/:id", authz.RequireAllMW("view_data"), h.GetInvestmentTradeByID)
	ap.GET("assets/:id/income", authz.RequireAllMW("view_data"), h.GetInvestmentIncomeByAsset)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertInvestmentAsset)
	ap.PUT("trades", authz.RequireAllMW("manage_data"), h.InsertInvestmentTrade)
	ap.PUT(":id", authz.RequireAllMW("manage_data"), h.UpdateInvestmentAsset)
	ap.PUT("trades/:id", authz.RequireAllMW("manage_data"), h.UpdateInvestmentTrade)
	ap.PUT("income", authz.RequireAllMW("manage_data"), h.CreateInvestmentIncome)
	ap.DELETE(":id", authz.RequireAllMW("manage_data"), h.DeleteInvestmentAsset)
	ap.DELETE("trades/:id", authz.RequireAllMW("manage_data"), h.DeleteInvestmentTrade)
	ap.DELETE("income/:id", authz.RequireAllMW("manage_data"), h.DeleteInvestmentIncome)
	ap.GET("tax-brackets", authz.RequireAllMW("view_data"), h.GetTaxBrackets)
	ap.PUT("tax-brackets", authz.RequireAllMW("manage_data"), h.InsertTaxBracket)
	ap.PUT("tax-brackets/:id", authz.RequireAllMW("manage_data"), h.UpdateTaxBracket)
	ap.DELETE("tax-brackets/:id", authz.RequireAllMW("manage_data"), h.DeleteTaxBracket)
	ap.POST("tax-brackets/copy", authz.RequireAllMW("manage_data"), h.CopyTaxBrackets)
	ap.GET("tax-settings", authz.RequireAllMW("view_data"), h.GetTaxSettings)
	ap.PUT("tax-settings", authz.RequireAllMW("manage_data"), h.SaveTaxSettings)
}

func (h *InvestmentHandler) GetInvestmentAssetsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	accountIDStr := qp.Get("account")
	var accountID *int64
	if accountIDStr != "" {
		id, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account id must be a valid integer", err))
			return
		}
		accountID = &id
	}

	records, paginator, err := h.Service.FetchInvestmentAssetsPaginated(ctx, userID, p, accountID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *InvestmentHandler) GetAllInvestmentAssets(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchAllInvestmentAssets(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *InvestmentHandler) GetInvestmentAssetByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	record, err := h.Service.FetchInvestmentAssetByID(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *InvestmentHandler) GetInvestmentTradesPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	assetIDStr := qp.Get("asset_id")
	var assetID *int64
	if assetIDStr != "" {
		id, err := strconv.ParseInt(assetIDStr, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "asset id must be a valid integer", err))
			return
		}
		assetID = &id
	}

	records, paginator, err := h.Service.FetchInvestmentTradesPaginated(ctx, userID, p, assetID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *InvestmentHandler) GetInvestmentTradeByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	record, err := h.Service.FetchInvestmentTradeByID(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *InvestmentHandler) InsertInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.InvestmentAssetReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.Service.InsertAsset(ctx, userID, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) InsertInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.InvestmentTradeReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.Service.InsertInvestmentTrade(ctx, userID, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) UpdateInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var record *models.InvestmentAssetReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.Service.UpdateInvestmentAsset(ctx, userID, id, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InvestmentHandler) UpdateInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var record *models.InvestmentTradeReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.Service.UpdateInvestmentTrade(ctx, userID, id, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteInvestmentAsset(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteInvestmentTrade(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) GetInvestmentIncomeByAsset(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	p := utils.GetPaginationParams(c.Request.URL.Query())

	records, paginator, err := h.Service.FetchInvestmentIncomeByAsset(ctx, userID, id, p)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	})
}

func (h *InvestmentHandler) CreateInvestmentIncome(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvestmentIncomeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.Service.CreateInvestmentIncome(ctx, userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteInvestmentIncome(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteInvestmentIncome(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) CopyTaxBrackets(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvestmentTaxBracketsCopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.CopyTaxBrackets(ctx, userID, req.FromType, req.ToType); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Brackets copied", "Success", http.StatusOK)
}

func (h *InvestmentHandler) GetTaxBrackets(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchTaxBrackets(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *InvestmentHandler) InsertTaxBracket(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvestmentTaxBracketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.Service.InsertTaxBracket(ctx, userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) UpdateTaxBracket(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var req models.InvestmentTaxBracketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.UpdateTaxBracket(ctx, userID, id, &req); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteTaxBracket(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteTaxBracket(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) GetTaxSettings(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.FetchTaxSettings(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *InvestmentHandler) SaveTaxSettings(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvestmentTaxSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.Service.SaveTaxSettings(ctx, userID, &req); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Settings saved", "Success", http.StatusOK)
}

func (h *InvestmentHandler) GetPortfolioAllocation(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.FetchPortfolioAllocation(ctx, userID, c.Query("currency"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *InvestmentHandler) GetPortfolioReturns(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.FetchPortfolioReturns(ctx, userID, c.Query("currency"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}
