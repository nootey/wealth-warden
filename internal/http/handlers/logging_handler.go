package handlers

import (
	"fmt"
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
	fmt.Println(queryParams)

	// Directly parse filters
	for _, field := range filterFields {
		// Check if multiple values exist (e.g., events[]=a&events[]=b)
		if values, ok := queryParams[field+"[]"]; ok {
			// If multiple values (array), use them directly
			if len(values) > 0 {
				filters[field] = values
			}
		} else if values, ok := queryParams[field]; ok {
			// If single value
			if len(values) > 0 {
				filters[field] = values[0]
			}
		}
	}

	fmt.Println(filters)

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
