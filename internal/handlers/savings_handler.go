package handlers

import "wealth-warden/internal/services"

type SavingsHandler struct {
	Service *services.SavingsService
}

func NewSavingsHandler(service *services.SavingsService) *SavingsHandler {
	return &SavingsHandler{Service: service}
}
