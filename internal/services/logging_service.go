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

func (s *LoggingService) FetchPaginatedLogs(c *gin.Context, tableName string) ([]map[string]interface{}, *utils.Paginator, error) {

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)

	fieldMappings := map[string]string{
		"categories": "category",
		"events":     "event",
		"causers":    "causer_id",
	}

	filters := make(map[string]interface{})
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

	dateStart := queryParams.Get("date_start")
	dateStop := queryParams.Get("date_stop")

	if dateStart != "" && dateStop != "" {
		filters["date_range"] = []string{dateStart, dateStop}
	}

	totalRecords, err := s.LoggingRepo.CountLogs(tableName, filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	logs, err := s.LoggingRepo.FindLogs(tableName, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, filters)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(logs)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  paginationParams.PageNumber,
		RowsPerPage:  paginationParams.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return logs, paginator, nil
}

func (s *LoggingService) FetchActivityLogFilterData(c *gin.Context, activityIndex string) (map[string]interface{}, error) {
	return s.LoggingRepo.FindActivityLogFilterData(activityIndex)
}
