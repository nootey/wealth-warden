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

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	series, err := h.Service.GetNetWorthSeries(ctx, userID, currency, r, from, to, accID)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *ChartingHandler) GetMonthlyCashFlowForYear(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	p := c.QueryMap("params")

	yearStr := c.Query("year")
	year, _ := strconv.Atoi(yearStr)

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

	series, err := h.Service.GetMonthlyCashFlowForYear(ctx, userID, year, accID)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *ChartingHandler) GetMonthlyCategoryBreakdown(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	p := c.QueryMap("params")

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

	catStr := c.Query("category")
	if catStr == "" {
		catStr = p["category"]
	}
	var catID *int64
	if strings.TrimSpace(catStr) != "" {
		v, err := strconv.ParseInt(catStr, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "category must be a valid integer", http.StatusBadRequest, err)
			return
		}
		catID = &v
	}

	class := c.DefaultQuery("class", "expense")
	asPercent := c.DefaultQuery("percent", "false") == "true"

	// Multi-year support via ?years=
	if ys := strings.TrimSpace(c.Query("years")); ys != "" {
		parts := strings.Split(ys, ",")
		if len(parts) > 5 {
			utils.ErrorMessage(c, "param error", "a maximum of 5 years is supported!", http.StatusBadRequest, nil)
			return
		}
		var years []int
		for _, s := range parts {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			yr, err := strconv.Atoi(s)
			if err != nil {
				utils.ErrorMessage(c, "param error", "years must be comma-separated integers", http.StatusBadRequest, err)
				return
			}
			years = append(years, yr)
		}
		if len(years) == 0 {
			utils.ErrorMessage(c, "param error", "years is empty", http.StatusBadRequest, nil)
			return
		}
		res, err := h.Service.GetCategoryUsageForYears(ctx, userID, years, class, accID, catID, asPercent)
		if err != nil {
			utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	yearStr := c.Query("year")
	if yearStr == "" {
		utils.ErrorMessage(c, "param error", "year is required when 'years' is not provided", http.StatusBadRequest, nil)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		utils.ErrorMessage(c, "param error", "year must be a valid integer", http.StatusBadRequest, err)
		return
	}

	series, err := h.Service.GetCategoryUsageForYear(ctx, userID, year, class, accID, catID, asPercent)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, series)
}
