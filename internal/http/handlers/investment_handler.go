package handlers

import (
	"net/http"
	"strconv"
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

func (h *InvestmentHandler) GetInvestmentHoldingsPaginated(c *gin.Context) {

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

	records, paginator, err := h.Service.FetchInvestmentHoldingsPaginated(ctx, userID, p, accountID)
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

func (h *InvestmentHandler) GetInvestmentTransactionsPaginated(c *gin.Context) {

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

	records, paginator, err := h.Service.FetchInvestmentTransactionsPaginated(ctx, userID, p, accountID)
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

func (h *InvestmentHandler) InsertInvestmentHolding(c *gin.Context) {

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
