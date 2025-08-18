package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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
	records, paginator, err := h.Service.FetchTransactionsPaginated(c)
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

	record, err := h.Service.FetchTransactionByID(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *TransactionHandler) GetCategories(c *gin.Context) {
	records, err := h.Service.FetchAllCategories(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) InsertTransaction(c *gin.Context) {

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

	if err := h.Service.InsertTransaction(c, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {

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

	if err := h.Service.UpdateTransaction(c, id, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}
