package handlers

import "wealth-warden/internal/services"

type BudgetHandler struct {
	Service *services.BudgetService
}

func NewBudgetHandler(service *services.BudgetService) *BudgetHandler {
	return &BudgetHandler{Service: service}
}
