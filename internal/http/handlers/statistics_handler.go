package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
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

func (h *StatisticsHandler) GetAccountBasicStatistics(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	idStr := c.Param("id")
	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	y := c.Query("year")
	if y == "" {
		utils.ErrorMessage(c, "param error", "year is required", http.StatusBadRequest, nil)
		return
	}
	year, err := strconv.Atoi(y)
	if err != nil || year < 1900 || year > 3000 {
		utils.ErrorMessage(c, "param error", "invalid year", http.StatusBadRequest, nil)
		return
	}

	stats, err := h.Service.GetAccountBasicStatistics(id, userID, year)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting basic statistics for account", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
