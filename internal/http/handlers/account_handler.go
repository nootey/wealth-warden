package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	service services.AccountServiceInterface
	v       validators.Validator
}

func NewAccountHandler(
	service services.AccountServiceInterface,
	v validators.Validator,
) *AccountHandler {
	return &AccountHandler{
		service: service,
		v:       v,
	}
}

func (h *AccountHandler) GetAccountsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeInactive := strings.EqualFold(qp.Get("inactive"), "true")
	classification := qp.Get("classification")

	records, paginator, err := h.service.FetchAccountsPaginated(ctx, userID, p, includeInactive, classification)
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

func (h *AccountHandler) GetAllAccounts(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	q := c.Request.URL.Query()
	includeInactive := strings.EqualFold(q.Get("inactive"), "true")

	records, err := h.service.FetchAllAccounts(ctx, userID, includeInactive)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountByID(c *gin.Context) {

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

	qp := c.Request.URL.Query()
	initialBalance := strings.EqualFold(qp.Get("initial_balance"), "true")

	records, err := h.service.FetchAccountByID(ctx, userID, id, initialBalance)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountByName(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	name := c.Param("name")
	if name == "" {
		err := errors.New("invalid name provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	records, err := h.service.FetchAccountByName(ctx, userID, name)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountTypes(c *gin.Context) {

	ctx := c.Request.Context()

	records, err := h.service.FetchAllAccountTypes(ctx)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountsBySubtype(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	sub := c.Param("sub")
	if sub == "" {
		err := errors.New("invalid subtype provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	records, err := h.service.FetchAccountsBySubtype(ctx, userID, sub)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountsByType(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	t := c.Param("type")
	if t == "" {
		err := errors.New("invalid type provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	records, err := h.service.FetchAccountsByType(ctx, userID, t)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) InsertAccount(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.AccountReq
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.service.InsertAccount(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)

}

func (h *AccountHandler) UpdateAccount(c *gin.Context) {

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

	var req *models.AccountReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.service.UpdateAccount(ctx, userID, id, req)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}

func (h *AccountHandler) ToggleAccountActiveState(c *gin.Context) {

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

	if err := h.service.ToggleAccountActiveState(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Toggle error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Account status updated", "Success", http.StatusOK)

}

func (h *AccountHandler) CloseAccount(c *gin.Context) {

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

	if err := h.service.CloseAccount(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *AccountHandler) BackfillBalancesForUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	from := c.Query("from")
	to := c.Query("to")

	if err := h.service.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
		utils.ErrorMessage(c, "Backfill error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Backfill completed", "Success", http.StatusOK)
}

func (h *AccountHandler) SaveAccountProjection(c *gin.Context) {

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

	var req *models.AccountProjectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	err = h.service.SaveAccountProjection(ctx, id, userID, req)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *AccountHandler) RevertAccountProjection(c *gin.Context) {

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

	err = h.service.RevertAccountProjection(ctx, id, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *AccountHandler) GetLatestBalance(c *gin.Context) {

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

	rec, err := h.service.FetchLatestBalance(ctx, id, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, rec)
}
