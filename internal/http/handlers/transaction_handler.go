package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
)

type TransactionHandler struct {
	Service *services.TransactionService
}

func NewTransactionHandler(service *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		Service: service,
	}
}

func (h *TransactionHandler) GetTransactionsPaginated(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

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

	records, paginator, err := h.Service.FetchTransactionsPaginated(userID, p, includeDeleted, accountID)
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

func (h *TransactionHandler) GetTransfersPaginated(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

	records, paginator, err := h.Service.FetchTransfersPaginated(userID, p, includeDeleted)
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

func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	idStr := c.Param("id")
	queryParams := c.Request.URL.Query()
	includeDeletedStr := queryParams.Get("deleted")

	includeDeleted := strings.EqualFold(includeDeletedStr, "true")

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

	record, err := h.Service.FetchTransactionByID(userID, id, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *TransactionHandler) GetCategories(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	q := c.Request.URL.Query()
	includeDeleted := strings.EqualFold(q.Get("deleted"), "true")

	records, err := h.Service.FetchAllCategories(userID, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) GetCategoryByID(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	idStr := c.Param("id")
	q := c.Request.URL.Query()
	includeDeleted := strings.EqualFold(q.Get("deleted"), "true")

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

	records, err := h.Service.FetchCategoryByID(userID, id, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) InsertTransaction(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var record *models.TransactionReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.InsertTransaction(userID, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) InsertTransfer(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var record *models.TransferReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.InsertTransfer(userID, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) InsertCategory(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var record *models.CategoryReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.InsertCategory(userID, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

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

	var record *models.TransactionReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdateTransaction(userID, id, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateCategory(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

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

	var record *models.CategoryReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.UpdateCategory(userID, id, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

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

	if err := h.Service.DeleteTransaction(userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteTransfer(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

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

	if err := h.Service.DeleteTransfer(userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteCategory(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

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

	if err := h.Service.DeleteCategory(userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreTransaction(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreTransaction(userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreCategory(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreCategory(userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreCategoryName(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreCategoryName(userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}
