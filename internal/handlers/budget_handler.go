package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type BudgetHandler struct {
	Service *services.BudgetService
}

func NewBudgetHandler(service *services.BudgetService) *BudgetHandler {
	return &BudgetHandler{Service: service}
}

func (h *BudgetHandler) GetCurrentMonthlyBudget(c *gin.Context) {
	record, err := h.Service.GetCurrentMonthlyBudget(c)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}
