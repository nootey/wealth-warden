package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
)

type OutflowHandler struct {
	Service *services.OutflowService
}

func NewOutflowHandler(service *services.OutflowService) *OutflowHandler {
	return &OutflowHandler{Service: service}
}

func (h *OutflowHandler) GetOutflowsPaginated(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)
	yearParam := queryParams.Get("year")

	outflows, totalRecords, err := h.Service.FetchOutflowsPaginated(c, paginationParams, yearParam)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	from := offset + 1
	if from > totalRecords {
		from = totalRecords
	}

	to := offset + len(outflows)
	if to > totalRecords {
		to = totalRecords
	}

	response := gin.H{
		"current_page":  paginationParams.PageNumber,
		"rows_per_page": paginationParams.RowsPerPage,
		"from":          from,
		"to":            to,
		"total_records": totalRecords,
		"data":          outflows,
	}

	c.JSON(http.StatusOK, response)
}

func (h *OutflowHandler) GetAllOutflowsGroupedByMonth(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	yearParam := queryParams.Get("year")

	records, err := h.Service.FetchAllOutflowsGroupedByMonth(c, yearParam)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *OutflowHandler) GetAllOutflowCategories(c *gin.Context) {
	outflowCategories, err := h.Service.FetchAllOutflowCategories(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, outflowCategories)
}

func (h *OutflowHandler) CreateNewOutflow(c *gin.Context) {

	var req validators.CreateOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	outflow := &models.Outflow{
		OutflowCategoryID: req.OutflowCategoryID,
		Amount:            req.Amount,
		OutflowDate:       req.OutflowDate,
		Description:       utils.CleanString(&req.Description).(*string),
	}

	if err := h.Service.CreateOutflow(c, outflow); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *OutflowHandler) UpdateOutflow(c *gin.Context) {

	var req validators.CreateOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.Outflow{
		ID:                req.ID,
		OutflowCategoryID: req.OutflowCategoryID,
		Amount:            req.Amount,
		OutflowDate:       req.OutflowDate,
		Description:       utils.CleanString(&req.Description).(*string),
	}

	if err := h.Service.UpdateOutflow(c, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *OutflowHandler) CreateNewReoccurringOutflow(c *gin.Context) {

	var req validators.ReoccurringOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.Outflow{
		OutflowCategoryID: req.Outflow.OutflowCategoryID,
		Amount:            req.Outflow.Amount,
		OutflowDate:       req.Outflow.OutflowDate,
	}

	var endDate *time.Time
	if req.RecOutflow.EndDate != nil {
		endDate = req.RecOutflow.EndDate
	} else {
		endDate = nil
	}

	recRecord := &models.RecurringAction{
		CategoryID:    req.Outflow.OutflowCategoryID,
		CategoryType:  req.RecOutflow.Category,
		Amount:        req.Outflow.Amount,
		StartDate:     req.RecOutflow.StartDate,
		EndDate:       endDate,
		IntervalUnit:  req.RecOutflow.IntervalUnit,
		IntervalValue: req.RecOutflow.IntervalValue,
	}

	if err := h.Service.CreateReoccurringOutflow(c, record, recRecord); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *OutflowHandler) CreateNewOutflowCategory(c *gin.Context) {

	var req validators.CreateOutflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.OutflowCategory{
		Name:          utils.CleanString(req.Name).(string),
		SpendingLimit: req.SpendingLimit,
		OutflowType:   utils.CleanString(req.OutflowType).(string),
	}

	if err := h.Service.CreateOutflowCategory(c, record); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *OutflowHandler) UpdateOutflowCategory(c *gin.Context) {

	var req validators.CreateOutflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.OutflowCategory{
		ID:            req.ID,
		Name:          utils.CleanString(req.Name).(string),
		SpendingLimit: req.SpendingLimit,
		OutflowType:   utils.CleanString(req.OutflowType).(string),
	}

	if err := h.Service.UpdateOutflowCategory(c, record); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *OutflowHandler) DeleteOutflow(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		err := errors.New("invalid request body")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteOutflow(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted", "Success", http.StatusOK)
}

func (h *OutflowHandler) DeleteOutflowCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		err := errors.New("invalid request body")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteOutflowCategory(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted", "Success", http.StatusOK)
}
