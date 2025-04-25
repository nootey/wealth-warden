package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type LoggingHandler struct {
	Service *services.LoggingService
}

func NewLoggingHandler(service *services.LoggingService) *LoggingHandler {
	return &LoggingHandler{
		Service: service,
	}
}

func (h *LoggingHandler) GetPaginatedLogs(c *gin.Context, tableName string, filterFields []string) {

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)

	filters := make(map[string]interface{})
	for _, field := range filterFields {
		if val := queryParams.Get(field); val != "" {
			filters[field] = val
		}
	}

	logs, totalRecords, err := h.Service.FetchPaginatedLogs(c, tableName, paginationParams, filters)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	from := offset + 1
	if from > totalRecords {
		from = totalRecords
	}

	to := offset + len(logs)
	if to > totalRecords {
		to = totalRecords
	}

	response := gin.H{
		"current_page":  paginationParams.PageNumber,
		"rows_per_page": paginationParams.RowsPerPage,
		"from":          from,
		"to":            to,
		"total_records": totalRecords,
		"data":          logs,
	}

	c.JSON(http.StatusOK, response)
}

func (h *LoggingHandler) GetAccessLogs(c *gin.Context) {
	h.GetPaginatedLogs(c, "access_logs", []string{"event", "service", "status", "ip_address", "causer_id"})
}

func (h *LoggingHandler) GetActivityLogs(c *gin.Context) {
	h.GetPaginatedLogs(c, "activity_logs", []string{"event", "category", "causer_id"})
}

func (h *LoggingHandler) GetNotificationLogs(c *gin.Context) {
	h.GetPaginatedLogs(c, "notification_logs", []string{"user_id", "type", "status", "destination"})
}
