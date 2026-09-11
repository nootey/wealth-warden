package handlers

import (
	"net/http"
	"strings"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type LoggingHandler struct {
	Service *services.LoggingService
}

func NewLoggingHandler(
	service *services.LoggingService,
) *LoggingHandler {
	return &LoggingHandler{
		Service: service,
	}
}

func (h *LoggingHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_activity_logs"), h.GetActivityLogs)
	apiGroup.GET("/filter-data", authz.RequireAllMW("view_activity_logs"), h.GetActivityLogFilterData)
	apiGroup.GET("/audit-trail", authz.RequireAllMW("view_data"), h.GetAuditTrail)
	apiGroup.DELETE("/:id", authz.RequireAllMW("delete_activity_logs"), h.DeleteActivityLog)
}

func (h *LoggingHandler) GetActivityLogs(c *gin.Context) {
	h.GetPaginatedLogs(c)
}

func (h *LoggingHandler) GetPaginatedLogs(c *gin.Context) {

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	ctx := c.Request.Context()

	records, paginator, err := h.Service.FetchPaginatedLogs(ctx, p)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *LoggingHandler) GetActivityLogFilterData(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	activityIndex := queryParams.Get("index")

	response, err := h.Service.FetchActivityLogFilterData(c, activityIndex)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (h *LoggingHandler) DeleteActivityLog(c *gin.Context) {

	ctx := c.Request.Context()

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteActivityLog(ctx, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *LoggingHandler) GetAuditTrail(c *gin.Context) {
	qp := c.Request.URL.Query()
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id := qp.Get("id")
	if id == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "id is required"))
		return
	}

	categoryStr := qp.Get("category")
	if categoryStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "category is required"))
		return
	}
	categories := strings.Split(categoryStr, ",")

	p := utils.GetPaginationParams(qp)

	eventStr := qp.Get("event")
	if eventStr == "" {
		c.JSON(http.StatusOK, gin.H{
			"current_page":  p.PageNumber,
			"rows_per_page": p.RowsPerPage,
			"total_records": 0,
			"from":          0,
			"to":            0,
			"data":          []models.ActivityLog{},
		})
		return
	}
	events := strings.Split(eventStr, ",")

	trail, paginator, err := h.Service.FetchAuditTrail(ctx, id, categories, events, userID, p)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"total_records": paginator.TotalRecords,
		"from":          paginator.From,
		"to":            paginator.To,
		"data":          trail,
	})
}
