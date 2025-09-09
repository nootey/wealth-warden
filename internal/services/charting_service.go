package services

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type ChartingService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.ChartingRepository
}

func NewChartingService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ChartingRepository,
) *ChartingService {
	return &ChartingService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}

func (s *ChartingService) GetNetWorthSeries(userID int64, currency, rangeKey, from, to string) ([]models.ChartPoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx := s.Repo.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Resolve date window
	var dfrom, dto time.Time
	var err error

	if from != "" || to != "" {
		if to == "" {
			dto = time.Now().UTC().Truncate(24 * time.Hour)
		} else {
			dto, err = time.Parse("2006-01-02", to)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("invalid to: %w", err)
			}
		}
		if from == "" {
			dfrom = dto.AddDate(0, 0, -30)
		} else {
			dfrom, err = time.Parse("2006-01-02", from)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("invalid from: %w", err)
			}
		}
	} else {
		// rangeKey -> window
		dto = time.Now().UTC().Truncate(24 * time.Hour)
		switch rangeKey {
		case "1w":
			dfrom = dto.AddDate(0, 0, -7)
		case "1m":
			dfrom = dto.AddDate(0, -1, 0)
		case "3m":
			dfrom = dto.AddDate(0, -3, 0)
		case "6m":
			dfrom = dto.AddDate(0, -6, 0)
		case "1y":
			dfrom = dto.AddDate(-1, 0, 0)
		case "5y":
			dfrom = dto.AddDate(-5, 0, 0)
		default:
			dfrom = dto.AddDate(0, 0, -30) // sensible default
		}
	}
	if dfrom.After(dto) {
		dfrom = dto
	}

	// Choose granularity
	days := int(dto.Sub(dfrom).Hours()/24) + 1
	gran := "day"
	if days > 90 && days <= 370 {
		gran = "week"
	}
	if days > 370 {
		gran = "month"
	}

	// Fetch
	points, err := s.Repo.FetchNetWorthSeries(tx, userID, currency, dfrom, dto, gran)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return points, nil
}
