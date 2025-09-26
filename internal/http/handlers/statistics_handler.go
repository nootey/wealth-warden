package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct {
	Service *services.StatisticsService
	v       *validators.GoValidator
}

func NewStatisticsHandler(
	service *services.StatisticsService,
	v *validators.GoValidator,
) *StatisticsHandler {
	return &StatisticsHandler{
		Service: service,
		v:       v,
	}
}

func (handler *StatisticsHandler) GetAccountBasicStatistics(c *gin.Context) {

}
