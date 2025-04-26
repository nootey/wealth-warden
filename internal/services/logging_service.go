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

func (s *LoggingService) FetchPaginatedLogs(c *gin.Context, tableName string, paginationParams utils.PaginationParams, filters map[string]interface{}) ([]map[string]interface{}, int, error) {

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
