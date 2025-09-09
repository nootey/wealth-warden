package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
)

type ChartingHandler struct {
	Service *services.ChartingService
	v       *validators.GoValidator
}

func NewChartingHandler(
	service *services.ChartingService,
	v *validators.GoValidator,
) *ChartingHandler {
	return &ChartingHandler{
		Service: service,
		v:       v,
	}
}

func (h *ChartingHandler) NetWorthChart(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	p := c.QueryMap("params")

	currency := c.Query("currency")
	if currency == "" {
		currency = p["currency"]
	}
	if currency == "" {
		currency = models.DefaultCurrency
	}

	r := strings.ToLower(strings.TrimSpace(c.Query("range")))
	if r == "" {
		r = strings.ToLower(strings.TrimSpace(p["range"]))
	}

	from := c.Query("from")
	if from == "" {
		from = p["from"]
	}

	to := c.Query("to")
	if to == "" {
		to = p["to"]
	}

	series, err := h.Service.GetNetWorthSeries(userID, currency, r, from, to)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currency": currency,
		"points":   series,
	})
}
