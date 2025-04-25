package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type ReoccurringActionHandler struct {
	Service *services.ReoccurringActionService
}

func NewReoccurringActionHandler(service *services.ReoccurringActionService) *ReoccurringActionHandler {
	return &ReoccurringActionHandler{
		Service: service,
	}
}

func (h *ReoccurringActionHandler) GetAllActionsForCategory(c *gin.Context) {

	categoryName := c.Query("categoryName")
	if categoryName == "" {
		err := errors.New("invalid category provided")
		utils.ErrorMessage(c, "Request error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	actions, err := h.Service.FetchAllActionsForCategory(c, categoryName)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, actions)
}

func (h *ReoccurringActionHandler) DeleteReoccurringAction(c *gin.Context) {

	var requestBody struct {
		ID           uint   `json:"id"`
		CategoryName string `json:"category_name"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.ErrorMessage(c, "Invalid request body", "Error", http.StatusBadRequest, err)
		return
	}

	err := h.Service.DeleteReoccurringAction(c, requestBody.ID, requestBody.CategoryName)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	utils.SuccessMessage(c, "Record has been deleted.", "Success", http.StatusOK)
}

func (h *ReoccurringActionHandler) GetAvailableRecordYears(c *gin.Context) {
	queryParams := c.Request.URL.Query()
	table := queryParams.Get("table")
	field := queryParams.Get("field")

	availableYears, err := h.Service.FetchAvailableYearsForRecords(c, table, field)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, availableYears)
}
