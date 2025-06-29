package handlers

import (
	"wealth-warden/internal/services"
)

type InvestmentsHandler struct {
	Service *services.InvestmentsService
}

func NewInvestmentHandler(service *services.InvestmentsService) *InvestmentsHandler {
	return &InvestmentsHandler{Service: service}
}
