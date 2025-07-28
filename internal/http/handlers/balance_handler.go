package handlers

import (
	"wealth-warden/internal/services"
)

type BalanceHandler struct {
	Service *services.BalanceService
}

func NewBalanceHandler(service *services.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		Service: service,
	}
}
