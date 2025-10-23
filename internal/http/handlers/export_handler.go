package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"
)

type ExportHandler struct {
	Service *services.ExportService
	v       *validators.GoValidator
}

func NewExportHandler(
	service *services.ExportService,
	v *validators.GoValidator,
) *ExportHandler {
	return &ExportHandler{
		Service: service,
		v:       v,
	}
}
