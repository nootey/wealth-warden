package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inflowTypes)
}

func (h *InflowHandler) CreateNewInflowType(c *gin.Context) {
	var inflowType *models.InflowType

	// Bind the JSON from the request to the inflowType struct
	if err := c.ShouldBindJSON(&inflowType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send the inflowType to your service
	if err := h.Service.CreateInflowType(inflowType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inflow type"})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Inflow type created successfully"})
}
