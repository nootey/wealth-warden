package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
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

	accStr := c.Query("account")
	if accStr == "" {
		accStr = p["account"]
	}

	var accID *int64
	if strings.TrimSpace(accStr) != "" {
		v, err := strconv.ParseInt(accStr, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "account must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	series, err := h.Service.GetNetWorthSeries(userID, currency, r, from, to, accID)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *ChartingHandler) GetMonthlyCashFlowForYear(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	p := c.QueryMap("params")

	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)

	accStr := c.Query("account")
	if accStr == "" {
		accStr = p["account"]
	}

	var accID *int64
	if strings.TrimSpace(accStr) != "" {
		v, err := strconv.ParseInt(accStr, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "account must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	series, err := h.Service.GetMonthlyCashFlowForYear(userID, year, accID)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, series)
}
