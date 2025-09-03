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

func (h *LoggingHandler) GetPaginatedLogs(c *gin.Context, tableName string, fieldMappings map[string]string) {

	records, paginator, err := h.Service.FetchPaginatedLogs(c, tableName, fieldMappings)
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

func (h *LoggingHandler) GetActivityLogs(c *gin.Context) {

	fieldMappings := map[string]string{
		"categories": "category",
		"events":     "event",
		"causers":    "causer_id",
	}

	h.GetPaginatedLogs(c, "activity_logs", fieldMappings)
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
