package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
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

func (h *BudgetHandler) CreateNewMonthlyBudget(c *gin.Context) {
	var req validators.CreateMonthlyBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	budgetReq := &models.MonthlyBudget{
		DynamicCategoryID: req.DynamicCategoryID,
	}

	budget, err := h.Service.CreateMonthlyBudget(c, budgetReq)
	if err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

func (h *BudgetHandler) CreateNewBudgetAllocation(c *gin.Context) {
	var req validators.CreateMonthlyBudgetAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(err.Error())(c, nil)
		return
	}

	budgetAllocReq := &models.MonthlyBudgetAllocation{
		MonthlyBudgetID:     req.MonthlyBudgetID,
		TotalAllocatedValue: req.Allocation,
		Category:            req.Category,
	}

	err := h.Service.CreateMonthlyBudgetAllocation(c, budgetAllocReq)
	if err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("Record created", "Success", http.StatusOK)(c.Writer, c.Request)
}

func (h *BudgetHandler) UpdateBudgetSnapshot(c *gin.Context) {

	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage("Invalid JSON", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	err := h.Service.UpdateBudgetSnapshot(c, req.ID)
	if err != nil {
		utils.ErrorMessage("Update error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage("New budget snapshot has been recorded", "Success", http.StatusOK)(c.Writer, c.Request)
}
