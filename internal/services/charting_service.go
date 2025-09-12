package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"sort"
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

func (s *ChartingService) GetNetWorthSeries(
	userID int64,
	currency,
	rangeKey,
	from, to string,
	accountID *int64,
) (*models.NetWorthResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
		case "ytd":
			dfrom = time.Date(dto.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		case "1y":
			dfrom = dto.AddDate(-1, 0, 0)
		case "5y":
			dfrom = dto.AddDate(-5, 0, 0)
		default:
			dfrom = dto.AddDate(0, -1, 0)
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

	points, err := s.Repo.FetchNetWorthSeries(tx, userID, currency, dfrom, dto, gran, accountID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// ensure ascending order by date
	sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })

	// keep the original number of real points
	origLen := len(points)

	// forward-fill so we end exactly at dto (and optionally start at dfrom)
	if origLen > 0 {
		last := points[origLen-1]
		// If last point isn't exactly dto, append a synthetic point at dto with last known value
		if !last.Date.Equal(dto) {
			points = append(points, models.ChartPoint{
				Date:  dto,
				Value: last.Value,
			})
		}
	}

	// latest snapshot
	curDate, curStr, err := s.Repo.FetchLatestNetWorth(tx, userID, currency, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// brand-new user: pretend current is zero as of dto
			curDate = dto
			curStr = "0"
		} else {
			tx.Rollback()
			return nil, err
		}
	}
	curDec, _ := decimal.NewFromString(curStr)

	// compute change: first vs last point in the chart window
	var prevEndDate, currentEndDate time.Time
	var prevEndVal, currentEndVal decimal.Decimal

	if len(points) > 0 {
		prevEndDate = points[0].Date
		prevEndVal = points[0].Value
		currentEndDate = points[len(points)-1].Date
		currentEndVal = points[len(points)-1].Value
	} else {
		// no points → treat as zero
		prevEndVal = decimal.Zero
		currentEndVal = decimal.Zero
	}

	abs := currentEndVal.Sub(prevEndVal)
	pct := decimal.Zero
	if !prevEndVal.IsZero() {
		pct = abs.Div(prevEndVal)
	}

	// special case: initial account balance spike
	if len(points) == 1 {
		// treat first ever balance as an increase from zero
		prevEndDate = points[0].Date
		prevEndVal = decimal.Zero
		currentEndDate = points[0].Date
		currentEndVal = points[0].Value
		abs = currentEndVal
		pct = decimal.NewFromInt(1) // 100% "gain"
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.NetWorthResponse{
		Currency: currency,
		Points:   points,
		Current:  models.ChartPoint{Date: curDate, Value: curDec},
		Change: &models.Change{
			PrevPeriodEndDate:  prevEndDate,
			PrevPeriodEndValue: prevEndVal,
			CurrentEndDate:     currentEndDate,
			CurrentEndValue:    currentEndVal,
			Abs:                abs,
			Pct:                pct,
		},
	}, nil
}
