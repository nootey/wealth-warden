package handlers

import (
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

func (h *StatisticsHandler) GetAvailableStatsYears(c *gin.Context) {

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

func (h *StatisticsHandler) GetCurrentMonthStats(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.GetCurrentMonthStats(ctx, userID, nil)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", "Error getting monthly stats", http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *StatisticsHandler) GetYearlyAverageForCategory(c *gin.Context) {

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
