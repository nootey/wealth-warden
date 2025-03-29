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
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
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
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *OutflowHandler) GetAllOutflowCategories(c *gin.Context) {
	outflowCategories, err := h.Service.FetchAllOutflowCategories(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, outflowCategories)
}

func (h *OutflowHandler) CreateNewOutflow(c *gin.Context) {

	var req validators.CreateOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	outflow := &models.Outflow{
		OutflowCategoryID: req.OutflowCategoryID,
		Amount:            req.Amount,
		OutflowDate:       req.OutflowDate,
		Description:       utils.CleanString(&req.Description).(*string),
	}

	if err := h.Service.CreateOutflow(c, outflow); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) UpdateOutflow(c *gin.Context) {

	var req validators.CreateOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
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
		utils.ErrorMessage("Update error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record updated", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) CreateNewReoccurringOutflow(c *gin.Context) {

	var req validators.ReoccurringOutflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
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
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) CreateNewOutflowCategory(c *gin.Context) {

	var req validators.CreateOutflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.OutflowCategory{
		Name:          utils.CleanString(req.Name).(string),
		SpendingLimit: req.SpendingLimit,
		OutflowType:   utils.CleanString(req.OutflowType).(string),
	}

	if err := h.Service.CreateOutflowCategory(c, record); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) UpdateOutflowCategory(c *gin.Context) {

	var req validators.CreateOutflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.OutflowCategory{
		ID:            req.ID,
		Name:          utils.CleanString(req.Name).(string),
		SpendingLimit: req.SpendingLimit,
		OutflowType:   utils.CleanString(req.OutflowType).(string),
	}

	if err := h.Service.UpdateOutflowCategory(c, record); err != nil {
		utils.ErrorMessage("Update error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record updated", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) DeleteOutflow(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage("Invalid request body", "Error", http.StatusBadRequest)(c, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteOutflow(c, id)
	if err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	utils.SuccessMessage("Record has been deleted", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *OutflowHandler) DeleteOutflowCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage("Invalid request body", "Error", http.StatusBadRequest)(c, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteOutflowCategory(c, id)
	if err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	utils.SuccessMessage("Record has been deleted", "Success", http.StatusOK)(c.Writer, c.Request)
}
