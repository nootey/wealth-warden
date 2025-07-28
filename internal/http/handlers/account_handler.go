package handlers

import (
	"wealth-warden/internal/services"
)

type AccountHandler struct {
	Service *services.AccountService
}

func NewAccountHandler(service *services.AccountService) *AccountHandler {
	return &AccountHandler{
		Service: service,
	}
}
