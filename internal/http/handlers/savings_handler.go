package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
)

type SavingsHandler struct {
	Service *services.SavingsService
}

func NewSavingsHandler(service *services.SavingsService) *SavingsHandler {
	return &SavingsHandler{Service: service}
}

func (h *SavingsHandler) GetSavingsPaginated(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)
	yearParam := queryParams.Get("year")

	records, totalRecords, err := h.Service.FetchSavingsPaginated(c, paginationParams, yearParam)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	from := offset + 1
	if from > totalRecords {
		from = totalRecords
	}

	to := offset + len(records)
	if to > totalRecords {
		to = totalRecords
	}

	response := gin.H{
		"current_page":  paginationParams.PageNumber,
		"rows_per_page": paginationParams.RowsPerPage,
		"from":          from,
		"to":            to,
		"total_records": totalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *SavingsHandler) GetAllSavingsGroupedByMonth(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	yearParam := queryParams.Get("year")

	records, err := h.Service.FetchAllSavingsGroupedByMonth(c, yearParam)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *SavingsHandler) GetAllSavingsCategories(c *gin.Context) {
	categories, err := h.Service.FetchAllSavingsCategories(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *SavingsHandler) CreateNewSavingsAllocation(c *gin.Context) {

	var req validators.CreateSavingsTransactionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.SavingsTransaction{
		ID:                req.ID,
		SavingsCategoryID: req.SavingsCategoryID,
		AllocatedAmount:   req.AllocatedAmount,
		TransactionDate:   req.TransactionDate,
		TransactionType:   req.TransactionType,
	}

	if err := h.Service.CreateSavingsAllocation(c, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *SavingsHandler) CreateNewSavingsDeduction(c *gin.Context) {

	var req validators.CreateSavingsTransactionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.SavingsTransaction{
		ID:                req.ID,
		SavingsCategoryID: req.SavingsCategoryID,
		AllocatedAmount:   req.AllocatedAmount,
		TransactionDate:   req.TransactionDate,
		TransactionType:   req.TransactionType,
	}

	if err := h.Service.CreateSavingsDeduction(c, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *SavingsHandler) CreateNewSavingsCategory(c *gin.Context) {

	var req validators.SavingsCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.SavingsCategory{
		Name:         utils.CleanString(req.Category.Name).(string),
		SavingsType:  utils.CleanString(req.Category.SavingsType).(string),
		GoalTarget:   req.Category.GoalTarget,
		AccountType:  utils.CleanString(req.Category.AccountType).(string),
		InterestRate: req.Category.InterestRate,
	}

	var recRecord *models.RecurringAction

	if *req.IsReoccurring {
		var endDate *time.Time
		if req.RecAction.EndDate != nil {
			endDate = req.RecAction.EndDate
		} else {
			endDate = nil
		}

		recRecord = &models.RecurringAction{
			CategoryID:    req.Category.ID,
			CategoryType:  req.RecAction.Category,
			Amount:        req.Allocation,
			StartDate:     req.RecAction.StartDate,
			EndDate:       endDate,
			IntervalUnit:  req.RecAction.IntervalUnit,
			IntervalValue: req.RecAction.IntervalValue,
		}
	}

	if err := h.Service.CreateSavingsCategory(c, record, recRecord); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *SavingsHandler) UpdateSavingsCategory(c *gin.Context) {

	var req validators.CreateSavingsCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.SavingsCategory{
		ID:           req.ID,
		Name:         utils.CleanString(req.Name).(string),
		SavingsType:  utils.CleanString(req.SavingsType).(string),
		GoalTarget:   req.GoalTarget,
		AccountType:  utils.CleanString(req.AccountType).(string),
		InterestRate: req.InterestRate,
	}

	if err := h.Service.UpdateSavingsCategory(c, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *SavingsHandler) DeleteSavingsCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage(c, "Invalid request body", "Error", http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteSavingsCategory(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted", "Success", http.StatusOK)
}
