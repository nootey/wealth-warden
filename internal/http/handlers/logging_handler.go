package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
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
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
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
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (h *LoggingHandler) DeleteActivityLog(c *gin.Context) {

	idStr := c.Param("id")
	ctx := c.Request.Context()

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

	if err := h.Service.DeleteActivityLog(ctx, id); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *LoggingHandler) GetAuditTrail(c *gin.Context) {
	qp := c.Request.URL.Query()
	ctx := c.Request.Context()

	id := qp.Get("id")
	if id == "" {
		err := errors.New("id is required")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	category := qp.Get("category")
	if category == "" {
		err := errors.New("category is required")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	eventStr := qp.Get("event")
	if eventStr == "" {
		c.JSON(http.StatusOK, gin.H{"data": []models.ActivityLog{}})
		return
	}
	events := strings.Split(eventStr, ",")

	trail, err := h.Service.FetchAuditTrail(ctx, id, category, events)
	if err != nil {
		utils.ErrorMessage(c, "Audit trail error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, trail)
}
