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

type InflowHandler struct {
	Service *services.InflowService
}

func NewInflowHandler(service *services.InflowService) *InflowHandler {
	return &InflowHandler{Service: service}
}

func (h *InflowHandler) GetInflowsPaginated(c *gin.Context) {

	records, paginator, err := h.Service.FetchInflowsPaginated(c)
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

func (h *InflowHandler) GetAllInflowsGroupedByMonth(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	yearParam := queryParams.Get("year")

	inflows, err := h.Service.FetchAllInflowsGroupedByMonth(c, yearParam)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, inflows)
}

func (h *InflowHandler) GetAllInflowCategories(c *gin.Context) {
	categories, err := h.Service.FetchAllInflowCategories(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *InflowHandler) GetAllDynamicCategories(c *gin.Context) {
	records, err := h.Service.FetchAllDynamicCategories(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *InflowHandler) CreateNewInflow(c *gin.Context) {

	var req validators.CreateInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	inflow := &models.Inflow{
		InflowCategoryID: req.InflowCategoryID,
		Amount:           req.Amount,
		InflowDate:       req.InflowDate,
		Description:      utils.CleanString(&req.Description).(*string),
	}

	if err := h.Service.CreateInflow(c, inflow); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InflowHandler) UpdateInflow(c *gin.Context) {

	var req validators.CreateInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	inflow := &models.Inflow{
		ID:               req.ID,
		InflowCategoryID: req.InflowCategoryID,
		Amount:           req.Amount,
		InflowDate:       req.InflowDate,
		Description:      utils.CleanString(&req.Description).(*string),
	}

	if err := h.Service.UpdateInflow(c, inflow); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InflowHandler) CreateNewReoccurringInflow(c *gin.Context) {

	var req validators.ReoccurringInflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	inflow := &models.Inflow{
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

	recInflow := &models.RecurringAction{
		CategoryID:    req.Inflow.InflowCategoryID,
		CategoryType:  req.RecInflow.Category,
		Amount:        req.Inflow.Amount,
		StartDate:     req.RecInflow.StartDate,
		EndDate:       endDate,
		IntervalUnit:  req.RecInflow.IntervalUnit,
		IntervalValue: req.RecInflow.IntervalValue,
	}

	if err := h.Service.CreateReoccurringInflow(c, inflow, recInflow); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InflowHandler) CreateNewInflowCategory(c *gin.Context) {

	var req validators.CreateInflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	inflowCategory := &models.InflowCategory{
		Name: utils.CleanString(req.Name).(string),
	}

	if err := h.Service.CreateInflowCategory(c, inflowCategory); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InflowHandler) CreateNewDynamicCategory(c *gin.Context) {

	var req validators.DynamicCategoryMapRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if len(req.Mapping.SecondaryLinks) > 0 && req.Mapping.SecondaryType == "" {
		err := errors.New("secondary type is required if secondary links are provided")
		utils.ErrorMessage(c, "Validation error", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	record := &models.DynamicCategory{
		Name: utils.CleanString(req.Category.Name).(string),
	}

	var mappings []models.DynamicCategoryMapping

	// Process Primary Links
	for _, item := range req.Mapping.PrimaryLinks {
		mapping := models.DynamicCategoryMapping{
			RelatedCategoryID:   item.ID,
			RelatedCategoryName: item.CategoryType,
		}
		mappings = append(mappings, mapping)
	}

	if len(req.Mapping.SecondaryLinks) > 0 {
		// Process Secondary Links
		for _, item := range req.Mapping.SecondaryLinks {

			mapping := models.DynamicCategoryMapping{
				RelatedCategoryID:   item.ID,
				RelatedCategoryName: req.Mapping.SecondaryType, // hardcoded for now, only outflows are supported as secondary links
			}
			mappings = append(mappings, mapping)
		}
	}

	err := h.Service.CreateDynamicCategoryWithMappings(c, record, mappings)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *InflowHandler) UpdateInflowCategory(c *gin.Context) {

	var req validators.CreateInflowCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	inflowCategory := &models.InflowCategory{
		ID:   req.ID,
		Name: utils.CleanString(req.Name).(string),
	}

	if err := h.Service.UpdateInflowCategory(c, inflowCategory); err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *InflowHandler) DeleteInflow(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		err := errors.New("invalid request body")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteInflow(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted.", "Success", http.StatusOK)
}

func (h *InflowHandler) DeleteInflowCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		err := errors.New("invalid request body")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteInflowCategory(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted", "Success", http.StatusOK)
}

func (h *InflowHandler) DeleteDynamicCategory(c *gin.Context) {

	var requestBody struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		err := errors.New("invalid request body")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	id := requestBody.ID

	err := h.Service.DeleteDynamicCategory(c, id)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted", "Success", http.StatusOK)
}
