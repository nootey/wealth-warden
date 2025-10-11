package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"
)

type ImportHandler struct {
	Service *services.ImportService
	v       *validators.GoValidator
}

func NewImportHandler(
	service *services.ImportService,
	v *validators.GoValidator,
) *ImportHandler {
	return &ImportHandler{
		Service: service,
		v:       v,
	}
}
