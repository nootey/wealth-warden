package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
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

func (h *InvestmentHandler) Routes(ap *gin.RouterGroup) {
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
	ap.GET("sync/:id", authz.RequireAllMW("view_data"), h.SyncAssetPNL)
	ap.GET("sync/account/:acc_id", authz.RequireAllMW("view_data"), h.SyncAssetAccountBalance)
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
			utils.ErrorMessage(c, "Error occurred", "account id must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accountID = &id
	}

	records, paginator, err := h.Service.FetchInvestmentAssetsPaginated(ctx, userID, p, accountID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
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
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *InvestmentHandler) GetInvestmentAssetByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	record, err := h.Service.FetchInvestmentAssetByID(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
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
			utils.ErrorMessage(c, "Error occurred", "asset id must be a valid integer", http.StatusBadRequest, err)
			return
		}
		assetID = &id
	}

	records, paginator, err := h.Service.FetchInvestmentTradesPaginated(ctx, userID, p, assetID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
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

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	record, err := h.Service.FetchInvestmentTradeByID(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *InvestmentHandler) InsertInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.InvestmentAssetReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertAsset(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) InsertInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.InvestmentTradeReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertInvestmentTrade(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InvestmentHandler) UpdateInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	var record *models.InvestmentAssetReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateInvestmentAsset(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InvestmentHandler) UpdateInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	var record *models.InvestmentTradeReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateInvestmentTrade(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteInvestmentAsset(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	if err := h.Service.DeleteInvestmentAsset(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) DeleteInvestmentTrade(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	if err := h.Service.DeleteInvestmentTrade(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *InvestmentHandler) SyncAssetPNL(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid asset id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	err = h.Service.RecalculateAssetPnL(ctx, id, userID)
	if err != nil {
		utils.ErrorMessage(c, "Asset PNL sync error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Asset sync started in the background.", "Pending", http.StatusOK)
}

func (h *InvestmentHandler) SyncAssetAccountBalance(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("acc_id")

	if idStr == "" {
		err := errors.New("invalid asset account id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	accID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	err = h.Service.RecalculateAccountBalances(ctx, accID, userID)
	if err != nil {
		utils.ErrorMessage(c, "Asset PNL sync error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Asset sync started in the background.", "Pending", http.StatusOK)
}
