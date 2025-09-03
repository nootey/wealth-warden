package services

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/models"
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

func (s *LoggingService) FetchPaginatedLogs(c *gin.Context) ([]models.ActivityLog, *utils.Paginator, error) {

	queryParams := c.Request.URL.Query()
	p := utils.GetPaginationParams(queryParams)

	totalRecords, err := s.LoggingRepo.CountLogs(p.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.LoggingRepo.FindLogs(offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *LoggingService) FetchActivityLogFilterData(c *gin.Context, activityIndex string) (map[string]interface{}, error) {
	return s.LoggingRepo.FindActivityLogFilterData(activityIndex)
}
