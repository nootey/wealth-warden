package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	Service services.AnalyticsServiceInterface
}

func NewAnalyticsHandler(service services.AnalyticsServiceInterface) *AnalyticsHandler {
	return &AnalyticsHandler{Service: service}
}

func (h *AnalyticsHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("/networth", authz.RequireAllMW("view_basic_statistics"), h.NetWorthChart)
	ap.GET("/asset/:id/chart", authz.RequireAllMW("view_basic_statistics"), h.AssetChart)
	ap.GET("/monthly-category-breakdown", authz.RequireAllMW("view_basic_statistics"), h.GetMonthlyCategoryBreakdown)
	ap.GET("/yearly-cash-flow-breakdown", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyCashFlowBreakdown)
	ap.GET("/sankey", authz.RequireAllMW("view_basic_statistics"), h.GetYearlySankeyData)
	ap.GET("/account", authz.RequireAllMW("view_basic_statistics"), h.GetAccountBasicStatistics)
	ap.GET("/breakdown/yearly", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyBreakdownStats)
	ap.GET("/years", authz.RequireAllMW("view_basic_statistics"), h.GetAvailableStatsYears)
	ap.GET("/month", authz.RequireAllMW("view_basic_statistics"), h.GetMonthlyStats)
	ap.GET("/today", authz.RequireAllMW("view_basic_statistics"), h.GetTodayStats)
	ap.GET("/categories/:id/average", authz.RequireAllMW("view_basic_statistics"), h.GetYearlyAverageForCategory)
	ap.GET("/reports", authz.RequireAllMW("view_basic_statistics"), h.ListReports)
	ap.POST("/reports/category", authz.RequireAllMW("view_basic_statistics"), h.GenerateCategoryReport)
	ap.GET("/reports/:id/download", authz.RequireAllMW("view_basic_statistics"), h.DownloadReport)
	ap.DELETE("/reports/:id", authz.RequireAllMW("manage_data"), h.DeleteReport)
}

func (h *AnalyticsHandler) NetWorthChart(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	p := c.QueryMap("params")

	currency := c.Query("currency")
	if currency == "" {
		currency = p["currency"]
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
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account must be a valid integer", err))
			return
		}
		accID = &v
	}

	series, err := h.Service.GetNetWorthSeries(ctx, userID, currency, r, from, to, accID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetYearlyCashFlowBreakdown(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	p := c.QueryMap("params")

	y := c.Query("year")
	if y == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "year is required"))
		return
	}
	year, err := strconv.Atoi(y)
	if err != nil || year < 1900 || year > 3000 {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid year"))
		return
	}

	accStr := c.Query("account")
	if accStr == "" {
		accStr = p["account"]
	}

	var accID *int64
	if strings.TrimSpace(accStr) != "" {
		v, err := strconv.ParseInt(accStr, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account must be a valid integer", err))
			return
		}
		accID = &v
	}

	series, err := h.Service.GetYearlyCashFlowBreakdown(ctx, userID, year, accID)
	if err != nil {
		_ = c.Error(err)
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
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account must be a valid integer", err))
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
			_ = c.Error(apperr.Wrap(apperr.Invalid, "category must be a valid integer", err))
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
			_ = c.Error(apperr.New(apperr.Invalid, "a maximum of 5 years is supported!"))
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
				_ = c.Error(apperr.Wrap(apperr.Invalid, "years must be comma-separated integers", err))
				return
			}
			years = append(years, yr)
		}
		if len(years) == 0 {
			_ = c.Error(apperr.New(apperr.Invalid, "years is empty"))
			return
		}
		res, err := h.Service.GetCategoryUsageForYears(ctx, userID, years, class, accID, catID, asPercent)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	yearStr := c.Query("year")
	if yearStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "year is required when 'years' is not provided"))
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "year must be a valid integer", err))
		return
	}

	series, err := h.Service.GetCategoryUsageForYear(ctx, userID, year, class, accID, catID, asPercent)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetYearlySankeyData(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	y := c.Query("year")
	if y == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "year is required"))
		return
	}
	year, err := strconv.Atoi(y)
	if err != nil || year < 1900 || year > 3000 {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid year"))
		return
	}

	accStr := c.Query("account")
	var accID *int64
	if strings.TrimSpace(accStr) != "" {
		v, err := strconv.ParseInt(accStr, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account must be a valid integer", err))
			return
		}
		accID = &v
	}

	sankeyData, err := h.Service.GetYearlySankeyData(ctx, userID, accID, year)
	if err != nil {
		_ = c.Error(err)
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
		_ = c.Error(apperr.New(apperr.Invalid, "year is required"))
		return
	}
	year, err := strconv.Atoi(y)
	if err != nil || year < 1900 || year > 3000 {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid year"))
		return
	}

	// accId (optional)
	var accID *int64
	if s := c.Query("acc_id"); s != "" && s != "null" && s != "undefined" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "accId must be a valid integer", err))
			return
		}
		accID = &v
	}

	stats, err := h.Service.GetAccountBasicStatistics(ctx, accID, userID, year)
	if err != nil {
		_ = c.Error(err)
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
			_ = c.Error(apperr.Wrap(apperr.Invalid, "accId must be a valid integer", err))
			return
		}
		accID = &v
	}

	includeMonths := c.Query("include_months") == "true"

	years, err := h.Service.GetAvailableStatsYears(ctx, accID, userID, includeMonths)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, years)
}

func (h *AnalyticsHandler) GetMonthlyStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if y := c.Query("year"); y != "" {
		v, err := strconv.Atoi(y)
		if err != nil || v < 1900 || v > 3000 {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "year must be a valid integer", err))
			return
		}
		year = v
	}

	if m := c.Query("month"); m != "" {
		v, err := strconv.Atoi(m)
		if err != nil || v < 1 || v > 12 {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "month must be between 1 and 12", err))
			return
		}
		month = v
	}

	records, err := h.Service.GetMonthlyStats(ctx, userID, nil, year, month)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *AnalyticsHandler) GetTodayStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.GetTodayStats(ctx, userID, nil)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *AnalyticsHandler) GetYearlyAverageForCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	categoryID, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var accountID int64
	if s := c.Query("account_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "account_id must be a valid integer", err))
			return
		}
		accountID = v
	} else {
		_ = c.Error(apperr.New(apperr.Invalid, "account_id is required"))
		return
	}

	isGroup := c.Query("is_group") == "true"

	average, err := h.Service.GetYearlyAverageForCategory(ctx, userID, accountID, categoryID, isGroup)
	if err != nil {
		_ = c.Error(err)
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
		_ = c.Error(apperr.New(apperr.Invalid, "year is required"))
		return
	}
	year, err := strconv.Atoi(y)
	if err != nil || year < 1900 || year > 3000 {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid year"))
		return
	}

	// accId (optional)
	var accID *int64
	if s := c.Query("acc_id"); s != "" && s != "null" && s != "undefined" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "accId must be a valid integer", err))
			return
		}
		accID = &v
	}

	var comparisonYear *int
	if cy := c.Query("comparison_year"); cy != "" && cy != "null" && cy != "undefined" {
		compYear, err := strconv.Atoi(cy)
		if err != nil || compYear < 1900 || compYear > 3000 {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "invalid comparison_year", err))
			return
		}
		comparisonYear = &compYear
	}

	stats, err := h.Service.GetYearlyBreakdownStats(ctx, accID, userID, year, comparisonYear)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) ListReports(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	p := utils.GetPaginationParams(c.Request.URL.Query())

	records, paginator, err := h.Service.ListReportsPaginated(ctx, userID, p)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          gin.H{"records": records},
	})
}

func (h *AnalyticsHandler) GenerateCategoryReport(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req struct {
		InflowCategoryIDs  []int64 `json:"inflow_category_ids"`
		OutflowCategoryIDs []int64 `json:"outflow_category_ids"`
		Years              []int   `json:"years"`
		Description        string  `json:"description"`
		AllTime            bool    `json:"all_time"`
		AccountID          *int64  `json:"account_id"`
		AccountTypeOnly    bool    `json:"account_type_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if !req.AllTime && len(req.Years) == 0 {
		_ = c.Error(apperr.New(apperr.Invalid, "years is required"))
		return
	}
	if len(req.Years) > 10 {
		_ = c.Error(apperr.New(apperr.Invalid, "a maximum of 10 years is supported"))
		return
	}
	if len(req.InflowCategoryIDs) == 0 && len(req.OutflowCategoryIDs) == 0 {
		_ = c.Error(apperr.New(apperr.Invalid, "at least one inflow or outflow category is required"))
		return
	}
	if req.AccountTypeOnly && req.AccountID == nil {
		_ = c.Error(apperr.New(apperr.Invalid, "an account is required when filtering by account type"))
		return
	}

	params := models.CategoryReportParams{
		InflowCategoryIDs:  req.InflowCategoryIDs,
		OutflowCategoryIDs: req.OutflowCategoryIDs,
		Years:              req.Years,
		Description:        strings.TrimSpace(req.Description),
		AllTime:            req.AllTime,
		AccountID:          req.AccountID,
		AccountTypeOnly:    req.AccountTypeOnly,
	}

	report, err := h.Service.GenerateCategoryReport(ctx, userID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) DownloadReport(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	data, name, err := h.Service.DownloadReport(ctx, id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	filename := fmt.Sprintf("%s.xlsx", name)
	mime := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, mime, data)
}

func (h *AnalyticsHandler) DeleteReport(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteReport(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *AnalyticsHandler) AssetChart(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	assetID, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	rangeKey := strings.ToLower(strings.TrimSpace(c.Query("range")))
	if rangeKey == "" {
		rangeKey = "ytd"
	}

	res, err := h.Service.FetchAssetChart(ctx, userID, assetID, rangeKey)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, res)
}
