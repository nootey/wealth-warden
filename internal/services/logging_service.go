package services

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type LoggingService struct {
	LoggingRepo *repositories.LoggingRepository
	Config      *config.Config
}

func NewLoggingService(cfg *config.Config, repo *repositories.LoggingRepository) *LoggingService {
	return &LoggingService{
		LoggingRepo: repo,
		Config:      cfg,
	}
}

func (s *LoggingService) FetchPaginatedLogs(c *gin.Context, tableName string, paginationParams utils.PaginationParams) ([]map[string]interface{}, int, error) {

	queryParams := c.Request.URL.Query()

	filters := make(map[string]interface{})
	fieldMappings := map[string]string{
		"categories": "category",
		"events":     "event",
		"causers":    "causer_id",
	}

	for queryField, dbField := range fieldMappings {
		if values, ok := queryParams[queryField+"[]"]; ok {
			if len(values) > 0 {
				filters[dbField] = values
			}
		} else if values, ok := queryParams[queryField]; ok {
			if len(values) > 0 {
				filters[dbField] = values[0]
			}
		}
	}

	totalRecords, err := s.LoggingRepo.CountLogs(tableName, filters)
	if err != nil {
		return nil, 0, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage

	logs, err := s.LoggingRepo.FindLogs(tableName, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, filters)
	if err != nil {
		return nil, 0, err
	}

	return logs, int(totalRecords), nil
}

func (s *LoggingService) FetchActivityLogFilterData(c *gin.Context, activityIndex string) (map[string]interface{}, error) {
	return s.LoggingRepo.FindActivityLogFilterData(activityIndex)
}
