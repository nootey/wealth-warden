package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
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

func (h *AnalyticsHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("/networth", authz.RequireAllMW("view_basic_statistics"), h.NetWorthChart)
	ap.GET("/monthly-category-breakdown", authz.RequireAllMW("view_basic_statistics"), h.GetMonthlyCategoryBreakdown)
	ap.GET("/yearly-cash-flow-breakdown", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyCashFlowBreakdown)
	ap.GET("/sankey", authz.RequireAllMW("view_basic_statistics"), h.GetYearlySankeyData)
	ap.GET("/account", authz.RequireAllMW("view_basic_statistics"), h.GetAccountBasicStatistics)
	ap.GET("/breakdown/yearly", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyBreakdownStats)
	ap.GET("/years", authz.RequireAllMW("view_basic_statistics"), h.GetAvailableStatsYears)
	ap.GET("/month", authz.RequireAllMW("view_basic_statistics"), h.GetCurrentMonthStats)
	ap.GET("/today", authz.RequireAllMW("view_basic_statistics"), h.GetTodayStats)
	ap.GET("/categories/:id/average", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyAverageForCategory)
}

func (h *AnalyticsHandler) NetWorthChart(c *gin.Context) {

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

func (h *AnalyticsHandler) GetYearlyCashFlowBreakdown(c *gin.Context) {

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

	series, err := h.Service.GetYearlyCashFlowBreakdown(ctx, userID, year, accID)
	if err != nil {
		utils.ErrorMessage(c, "Failed to load chart", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetMonthlyCategoryBreakdown(c *gin.Context) {

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

func (h *AnalyticsHandler) GetYearlySankeyData(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	accStr := c.Query("account")
	var accID *int64
	if strings.TrimSpace(accStr) != "" {
		v, err := strconv.ParseInt(accStr, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "account must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	sankeyData, err := h.Service.GetYearlySankeyData(ctx, userID, accID, year)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting sankey data", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, sankeyData)
}

func (h *AnalyticsHandler) GetAccountBasicStatistics(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	// year (required)
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

	// accId (optional)
	var accID *int64
	if s := c.Query("acc_id"); s != "" && s != "null" && s != "undefined" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "accId must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	stats, err := h.Service.GetAccountBasicStatistics(ctx, accID, userID, year)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting basic statistics for account", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) GetAvailableStatsYears(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var accID *int64
	if s := c.Query("acc_id"); s != "" && s != "null" && s != "undefined" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "accId must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	years, err := h.Service.GetAvailableStatsYears(ctx, accID, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting available years", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, years)
}

func (h *AnalyticsHandler) GetCurrentMonthStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.GetCurrentMonthStats(ctx, userID, nil)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting monthly stats", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *AnalyticsHandler) GetTodayStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.GetTodayStats(ctx, userID, nil)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting todays stats", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *AnalyticsHandler) GetYearlyAverageForCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var categoryID int64
	if s := c.Param("id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "id must be a valid integer", http.StatusBadRequest, err)
			return
		}
		categoryID = v
	} else {
		utils.ErrorMessage(c, "param error", "id is required", http.StatusBadRequest, nil)
		return
	}

	var accountID int64
	if s := c.Query("account_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "account_id must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accountID = v
	} else {
		utils.ErrorMessage(c, "param error", "account_id is required", http.StatusBadRequest, nil)
		return
	}

	isGroup := c.Query("is_group") == "true"

	average, err := h.Service.GetYearlyAverageForCategory(ctx, userID, accountID, categoryID, isGroup)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting yearly average", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"average": average})
}

func (h *AnalyticsHandler) GetYearlyBreakdownStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	// year (required)
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

	// accId (optional)
	var accID *int64
	if s := c.Query("acc_id"); s != "" && s != "null" && s != "undefined" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "param error", "accId must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accID = &v
	}

	var comparisonYear *int
	if cy := c.Query("comparison_year"); cy != "" && cy != "null" && cy != "undefined" {
		compYear, err := strconv.Atoi(cy)
		if err != nil || compYear < 1900 || compYear > 3000 {
			utils.ErrorMessage(c, "param error", "invalid comparison_year", http.StatusBadRequest, err)
			return
		}
		comparisonYear = &compYear
	}

	stats, err := h.Service.GetYearlyBreakdownStats(ctx, accID, userID, year, comparisonYear)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting breakdown", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
