package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"
)

type InvestmentHandler struct {
	Service *services.InvestmentService
	v       *validators.GoValidator
}

func NewInvestmentHandler(
	service *services.InvestmentService,
	v *validators.GoValidator,
) *InvestmentHandler {
	return &InvestmentHandler{
		Service: service,
		v:       v,
	}
}
