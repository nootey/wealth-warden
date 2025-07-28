package handlers

import (
	"wealth-warden/internal/services"
)

type CategoryHandler struct {
	Service *services.CategoryService
}

func NewCategoryHandler(service *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		Service: service,
	}
}
