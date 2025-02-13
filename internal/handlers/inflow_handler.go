package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
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

	inflows, totalRecords, err := h.Service.GetInflowsPaginated(paginationParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching inflows"})
		return
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	from := offset + 1
	to := offset + len(inflows)

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

func (h *InflowHandler) GetAllInflowCategories(c *gin.Context) {
	inflowCategories, err := h.Service.FetchAllInflowCategories()
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, inflowCategories)
}

func (h *InflowHandler) CreateNewInflow(c *gin.Context) {
	var inflow *models.Inflow

	if err := c.ShouldBindJSON(&inflow); err != nil {
		utils.ErrorMessage("Json bind error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	if err := h.Service.CreateInflow(inflow); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("", "Inflow created successfully", http.StatusOK)(c.Writer, c.Request)
}

func (h *InflowHandler) CreateNewInflowCategory(c *gin.Context) {
	var inflowCategory *models.InflowCategory

	if err := c.ShouldBindJSON(&inflowCategory); err != nil {
		utils.ErrorMessage("Json bind error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	if err := h.Service.CreateInflowCategory(inflowCategory); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage(inflowCategory.Name, "Inflow category created successfully", http.StatusOK)(c.Writer, c.Request)
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

	utils.SuccessMessage("Inflow has been deleted successfully.", "Success", http.StatusOK)(c.Writer, c.Request)
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

	utils.SuccessMessage("Inflow category has been deleted successfully.", "Success", http.StatusOK)(c.Writer, c.Request)
}
