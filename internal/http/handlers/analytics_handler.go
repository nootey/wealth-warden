package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	Service services.AnalyticsServiceInterface
	v       validators.Validator
}

func NewAnalyticsHandler(service services.AnalyticsServiceInterface, v validators.Validator) *AnalyticsHandler {
	return &AnalyticsHandler{Service: service, v: v}
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

func (h *AnalyticsHandler) bindQuery(c *gin.Context, q any) bool {
	if err := c.ShouldBindQuery(q); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid query parameters", err))
		return false
	}

	if err := h.v.ValidateStruct(q); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return false
	}

	return true
}

func (h *AnalyticsHandler) NetWorthChart(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.NetWorthQuery
	if !h.bindQuery(c, &q) {
		return
	}
	r := strings.ToLower(strings.TrimSpace(q.Range))

	series, err := h.Service.GetNetWorthSeries(ctx, userID, q.Currency, r, q.From, q.To, q.Account)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetYearlyCashFlowBreakdown(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.YearAccountQuery
	if !h.bindQuery(c, &q) {
		return
	}

	series, err := h.Service.GetYearlyCashFlowBreakdown(ctx, userID, q.Year, q.Account)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetMonthlyCategoryBreakdown(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.CategoryBreakdownQuery
	if !h.bindQuery(c, &q) {
		return
	}

	if len(q.Years) > 0 {
		res, err := h.Service.GetCategoryUsageForYears(ctx, userID, q.Years, q.Class, q.Account, q.Category, q.Percent)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	series, err := h.Service.GetCategoryUsageForYear(ctx, userID, q.Year, q.Class, q.Account, q.Category, q.Percent)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, series)
}

func (h *AnalyticsHandler) GetYearlySankeyData(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.YearAccountQuery
	if !h.bindQuery(c, &q) {
		return
	}

	sankeyData, err := h.Service.GetYearlySankeyData(ctx, userID, q.Account, q.Year)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sankeyData)
}

func (h *AnalyticsHandler) GetAccountBasicStatistics(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.AccountStatsQuery
	if !h.bindQuery(c, &q) {
		return
	}

	stats, err := h.Service.GetAccountBasicStatistics(ctx, q.AccID, userID, q.Year)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) GetAvailableStatsYears(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.StatsYearsQuery
	if !h.bindQuery(c, &q) {
		return
	}

	years, err := h.Service.GetAvailableStatsYears(ctx, q.AccID, userID, q.IncludeMonths)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, years)
}

func (h *AnalyticsHandler) GetMonthlyStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.MonthlyStatsQuery
	if !h.bindQuery(c, &q) {
		return
	}

	now := time.Now()
	if q.Year == 0 {
		q.Year = now.Year()
	}
	if q.Month == 0 {
		q.Month = int(now.Month())
	}

	records, err := h.Service.GetMonthlyStats(ctx, userID, nil, q.Year, q.Month)
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

	var q models.CategoryAverageQuery
	if !h.bindQuery(c, &q) {
		return
	}

	average, err := h.Service.GetYearlyAverageForCategory(ctx, userID, q.AccountID, categoryID, q.IsGroup)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"average": average})
}

func (h *AnalyticsHandler) GetYearlyBreakdownStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var q models.YearlyBreakdownQuery
	if !h.bindQuery(c, &q) {
		return
	}

	stats, err := h.Service.GetYearlyBreakdownStats(ctx, q.AccID, userID, q.Year, q.ComparisonYear)
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

	var req models.CategoryReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	// the validator's `required` accepts an empty slice, so non-empty checks stay here
	if !req.AllTime && len(req.Years) == 0 {
		_ = c.Error(apperr.New(apperr.Invalid, "years is required"))
		return
	}
	if len(req.InflowCategoryIDs) == 0 && len(req.OutflowCategoryIDs) == 0 {
		_ = c.Error(apperr.New(apperr.Invalid, "at least one inflow or outflow category is required"))
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
