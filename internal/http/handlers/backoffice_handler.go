package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type BackofficeHandler struct {
	Service *services.BackofficeService
	v       *validators.GoValidator
}

func NewBackofficeHandler(
	service *services.BackofficeService,
	v *validators.GoValidator,
) *BackofficeHandler {
	return &BackofficeHandler{
		Service: service,
		v:       v,
	}
}

func (h *BackofficeHandler) Routes(ap *gin.RouterGroup) {

}
