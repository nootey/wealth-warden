package services

import (
	"context"
	"fmt"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type LoggingServiceInterface interface {
	FetchPaginatedLogs(ctx context.Context, p utils.PaginationParams) ([]models.ActivityLog, *utils.Paginator, error)
	FetchActivityLogFilterData(ctx context.Context, activityIndex string) (map[string]interface{}, error)
	DeleteActivityLog(ctx context.Context, id int64) error
}

type LoggingService struct {
	cfg  *config.Config
	repo repositories.LoggingRepositoryInterface
}

func NewLoggingService(
	cfg *config.Config,
	repo *repositories.LoggingRepository,
) *LoggingService {
	return &LoggingService{
		repo: repo,
		cfg:  cfg,
	}
}

var _ LoggingServiceInterface = (*LoggingService)(nil)

func (s *LoggingService) FetchPaginatedLogs(ctx context.Context, p utils.PaginationParams) ([]models.ActivityLog, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountLogs(ctx, p.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindLogs(ctx, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters)
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

func (s *LoggingService) FetchActivityLogFilterData(ctx context.Context, activityIndex string) (map[string]interface{}, error) {
	return s.repo.FindActivityLogFilterData(ctx, activityIndex)
}

func (s *LoggingService) DeleteActivityLog(ctx context.Context, id int64) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load the log to confirm existence
	tr, err := s.repo.FindActivityLogByID(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("can't find log with given id %w", err)
	}

	// Delete log
	if err := s.repo.DeleteActivityLog(ctx, tx, tr.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
