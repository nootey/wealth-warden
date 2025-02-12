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

func (h *InflowHandler) GetAllInflowTypes(c *gin.Context) {
	inflowTypes, err := h.Service.FetchAllInflowTypes()
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, inflowTypes)
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

func (h *InflowHandler) CreateNewInflowType(c *gin.Context) {
	var inflowType *models.InflowType

	if err := c.ShouldBindJSON(&inflowType); err != nil {
		utils.ErrorMessage("Json bind error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	if err := h.Service.CreateInflowType(inflowType); err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage(inflowType.Name, "Inflow type created successfully", http.StatusOK)(c.Writer, c.Request)
}
