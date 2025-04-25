package handlers

import (
	"errors"
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
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *BudgetHandler) CreateNewMonthlyBudget(c *gin.Context) {
	var req validators.CreateMonthlyBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	budgetReq := &models.MonthlyBudget{
		DynamicCategoryID: req.DynamicCategoryID,
	}

	budget, err := h.Service.CreateMonthlyBudget(c, budgetReq)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

func (h *BudgetHandler) CreateNewBudgetAllocation(c *gin.Context) {
	var req validators.CreateMonthlyBudgetAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	budgetAllocReq := &models.MonthlyBudgetAllocation{
		MonthlyBudgetID: req.MonthlyBudgetID,
		Method:          req.Method,
		Allocation:      req.Allocation,
		AllocatedValue:  req.AllocatedValue,
		Category:        req.Category,
	}

	err := h.Service.CreateMonthlyBudgetAllocation(c, budgetAllocReq)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *BudgetHandler) UpdateMonthlyBudget(c *gin.Context) {

	var req validators.UpdateMonthlyBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	budget := &models.MonthlyBudgetUpdate{
		ID: req.BudgetID,
	}

	switch req.Field {
	case "budget_snapshot":
		value, ok := req.Value.(float64)
		if !ok {
			err := errors.New("expected a float64")
			utils.ErrorMessage(c, "Invalid value type", err.Error(), http.StatusBadRequest, err)
			return
		}
		budget.BudgetSnapshot = &value
	case "snapshot_threshold":
		value, ok := req.Value.(float64)
		if !ok {
			err := errors.New("expected a float64")
			utils.ErrorMessage(c, "Invalid value type", err.Error(), http.StatusBadRequest, err)
			return
		}
		budget.SnapshotThreshold = &value
	}

	err := h.Service.UpdateMonthlyBudget(c, budget)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Budget has been updated", "Success", http.StatusOK)
}

func (h *BudgetHandler) SynchronizeCurrentMonthlyBudget(c *gin.Context) {

	err := h.Service.SynchronizeCurrentMonthlyBudget(c)
	if err != nil {
		utils.ErrorMessage(c, "Sync error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Monthly budget has been synchronized!", "Success", http.StatusOK)
}

func (h *BudgetHandler) SynchronizeCurrentMonthlyBudgetSnapshot(c *gin.Context) {

	err := h.Service.SynchronizeCurrentMonthlyBudgetSnapshot(c)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "New budget snapshot has been recorded", "Success", http.StatusOK)
}
