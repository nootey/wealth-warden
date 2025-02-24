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

	outflows, totalRecords, err := h.Service.FetchOutflowsPaginated(c, paginationParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching outflows"})
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

	outflows, err := h.Service.FetchAllOutflowsGroupedByMonth(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, outflows)
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
	}

	if err := h.Service.CreateOutflow(c, outflow); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("", "outflow created successfully", http.StatusOK)(c.Writer, c.Request)
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

	outflow := &models.Outflow{
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

	recOutflow := &models.RecurringAction{
		CategoryID:    req.Outflow.OutflowCategoryID,
		CategoryType:  req.RecOutflow.Category,
		Amount:        req.Outflow.Amount,
		StartDate:     req.RecOutflow.StartDate,
		EndDate:       endDate,
		IntervalUnit:  req.RecOutflow.IntervalUnit,
		IntervalValue: req.RecOutflow.IntervalValue,
	}

	if err := h.Service.CreateReoccurringOutflow(c, outflow, recOutflow); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("", "Reoccurring outflow created successfully", http.StatusOK)(c.Writer, c.Request)
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

	outflowCategory := &models.OutflowCategory{
		Name: req.Name,
	}

	if err := h.Service.CreateOutflowCategory(c, outflowCategory); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage(outflowCategory.Name, "Outflow category created successfully", http.StatusOK)(c.Writer, c.Request)
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

	utils.SuccessMessage("Outflow has been deleted successfully.", "Success", http.StatusOK)(c.Writer, c.Request)
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

	utils.SuccessMessage("Outflow category has been deleted successfully.", "Success", http.StatusOK)(c.Writer, c.Request)
}
