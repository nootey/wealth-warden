package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	Service *services.AnalyticsService
	v       *validators.GoValidator
}

func NewAnalyticsHandler(
	service *services.AnalyticsService,
	v *validators.GoValidator,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		Service: service,
		v:       v,
	}
}

func (h *AnalyticsHandler) Routes(apiGroup *gin.RouterGroup) {

}
