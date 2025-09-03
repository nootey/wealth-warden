package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type LoggingService struct {
	Repo   *repositories.LoggingRepository
	Config *config.Config
}

func NewLoggingService(cfg *config.Config, repo *repositories.LoggingRepository) *LoggingService {
	return &LoggingService{
		Repo:   repo,
		Config: cfg,
	}
}

func (s *LoggingService) FetchPaginatedLogs(c *gin.Context) ([]models.ActivityLog, *utils.Paginator, error) {

	queryParams := c.Request.URL.Query()
	p := utils.GetPaginationParams(queryParams)

	totalRecords, err := s.Repo.CountLogs(p.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindLogs(offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters)
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
	return s.Repo.FindActivityLogFilterData(activityIndex)
}

func (s *LoggingService) DeleteActivityLog(c *gin.Context, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the log to confirm existence
	tr, err := s.Repo.FindActivityLogByID(tx, id)
	if err != nil {
		return fmt.Errorf("can't find log with given id %w", err)
	}

	// Delete log
	if err := s.Repo.DeleteActivityLog(tx, tr.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
