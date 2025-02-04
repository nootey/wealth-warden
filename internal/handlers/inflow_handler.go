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

func (h *InflowHandler) GetAllInflowTypes(c *gin.Context) {
	inflowTypes, err := h.Service.FetchAllInflowTypes()
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, inflowTypes)
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

	utils.SuccessMessage(inflowType.Name, "Inflow created successfully", http.StatusOK)(c.Writer, c.Request)
}
