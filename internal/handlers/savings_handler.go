package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
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

func (h *SavingsHandler) GetAllSavingsCategories(c *gin.Context) {
	categories, err := h.Service.FetchAllSavingsCategories(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *SavingsHandler) CreateNewSavingsCategory(c *gin.Context) {

	var req validators.CreateSavingsCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	record := &models.SavingsCategory{
		Name:         utils.CleanString(req.Name).(string),
		SavingsType:  req.SavingsType,
		GoalValue:    req.GoalValue,
		AccountType:  req.AccountType,
		InterestRate: req.InterestRate,
	}

	if err := h.Service.CreateSavingsCategory(c, record); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record created", "Success", http.StatusOK)(c.Writer, c.Request)
}
