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

type InflowHandler struct {
	Service *services.InflowService
}

func NewInflowHandler(service *services.InflowService) *InflowHandler {
	return &InflowHandler{Service: service}
}

func (h *InflowHandler) GetInflowsPaginated(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)

	inflows, totalRecords, err := h.Service.FetchInflowsPaginated(c, paginationParams)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	from := offset + 1
	if from > totalRecords {
		from = totalRecords
	}

	to := offset + len(inflows)
	if to > totalRecords {
		to = totalRecords
	}

	response := gin.H{
		"current_page":  paginationParams.PageNumber,
		"rows_per_page": paginationParams.RowsPerPage,
		"from":          from,
		"to":            to,
		"total_records": totalRecords,
		"data":          inflows,
	}

	c.JSON(http.StatusOK, response)
}

func (h *InflowHandler) GetAllInflowsGroupedByMonth(c *gin.Context) {

	inflows, err := h.Service.FetchAllInflowsGroupedByMonth(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, inflows)
}

func (h *InflowHandler) GetAllInflowCategories(c *gin.Context) {
	inflowCategories, err := h.Service.FetchAllInflowCategories(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, inflowCategories)
}

func (h *InflowHandler) CreateNewInflow(c *gin.Context) {

	var req validators.CreateInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.Inflow{
		InflowCategoryID: req.InflowCategoryID,
		Amount:           req.Amount,
		InflowDate:       req.InflowDate,
		Description:      &req.Description,
	}

	if err := h.Service.CreateInflow(c, record); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Inflow created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) UpdateInflow(c *gin.Context) {

	var req validators.CreateInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.Inflow{
		ID:               req.ID,
		InflowCategoryID: req.InflowCategoryID,
		Amount:           req.Amount,
		InflowDate:       req.InflowDate,
		Description:      &req.Description,
	}

	if err := h.Service.UpdateInflow(c, record); err != nil {
		utils.ErrorMessage("Update error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Inflow updated", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) CreateNewReoccurringInflow(c *gin.Context) {

	var req validators.ReoccurringInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.Inflow{
		InflowCategoryID: req.Inflow.InflowCategoryID,
		Amount:           req.Inflow.Amount,
		InflowDate:       req.Inflow.InflowDate,
	}

	var endDate *time.Time
	if req.RecInflow.EndDate != nil {
		endDate = req.RecInflow.EndDate
	} else {
		endDate = nil
	}

	recRecord := &models.RecurringAction{
		CategoryID:    req.Inflow.InflowCategoryID,
		CategoryType:  req.RecInflow.Category,
		Amount:        req.Inflow.Amount,
		StartDate:     req.RecInflow.StartDate,
		EndDate:       endDate,
		IntervalUnit:  req.RecInflow.IntervalUnit,
		IntervalValue: req.RecInflow.IntervalValue,
	}

	if err := h.Service.CreateReoccurringInflow(c, record, recRecord); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Reoccurring inflow created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) CreateNewInflowCategory(c *gin.Context) {

	var req validators.CreateInflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.InflowCategory{
		Name: req.Name,
	}

	if err := h.Service.CreateInflowCategory(c, record); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Inflow category created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) UpdateInflowCategory(c *gin.Context) {

	var req validators.CreateInflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.InflowCategory{
		ID:   req.ID,
		Name: req.Name,
	}

	if err := h.Service.UpdateInflowCategory(c, record); err != nil {
		utils.ErrorMessage("Update error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Inflow category updated", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) DeleteInflow(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage("Invalid request body", "Error", http.StatusBadRequest)(c, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteInflow(c, id)
	if err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	utils.SuccessMessage("Inflow has been deleted.", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) DeleteInflowCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage("Invalid request body", "Error", http.StatusBadRequest)(c, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteInflowCategory(c, id)
	if err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	utils.SuccessMessage("Inflow category has been deleted", "Success", http.StatusOK)(c.Writer, c.Request)
}
