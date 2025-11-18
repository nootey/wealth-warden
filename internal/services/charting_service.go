package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"

	"github.com/shopspring/decimal"
)

type ChartingService struct {
	Config  *config.Config
	Ctx     *DefaultServiceContext
	Repo    *repositories.ChartingRepository
	AccRepo *repositories.AccountRepository
	TxRepo  *repositories.TransactionRepository
}

func NewChartingService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ChartingRepository,
	accRepo *repositories.AccountRepository,
	txRepo *repositories.TransactionRepository,
) *ChartingService {
	return &ChartingService{
		Ctx:     ctx,
		Config:  cfg,
		Repo:    repo,
		AccRepo: accRepo,
		TxRepo:  txRepo,
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
		prevEndVal = decimal.Zero
		currentEndVal = decimal.Zero
	}

	abs := currentEndVal.Sub(prevEndVal)
	pct := decimal.Zero
	if !prevEndVal.IsZero() {
		pct = abs.Div(prevEndVal)
	}

	// special case: initial account balance spike (only one real bucket in window)
	if origLen == 1 {
		prevEndDate = points[0].Date
		prevEndVal = decimal.Zero
		currentEndDate = points[len(points)-1].Date
		currentEndVal = points[len(points)-1].Value
		abs = currentEndVal
		pct = decimal.NewFromInt(1) // 100% gain
	}

	var acc *models.Account
	if accountID != nil {
		acc, err = s.AccRepo.FindAccountByID(tx, *accountID, userID, false)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	nwRes := &models.NetWorthResponse{
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
	}

	if acc != nil {
		nwRes.AssetType = &acc.AccountType.Classification
	}

	return nwRes, nil
}

func (s *ChartingService) GetMonthlyCashFlowForYear(userID int64, year int, accountID *int64) (*models.MonthlyCashflowResponse, error) {
	txs, err := s.TxRepo.GetTransactionsForYear(userID, year, accountID)
	if err != nil {
		return nil, err
	}

	months := make(map[int]*models.MonthlyCashflow)
	for m := 1; m <= 12; m++ {
		months[m] = &models.MonthlyCashflow{
			Month:    m,
			Inflows:  decimal.NewFromInt(0),
			Outflows: decimal.NewFromInt(0),
			Net:      decimal.NewFromInt(0),
		}
	}

	for _, tx := range txs {
		month := int(tx.TxnDate.Month())
		switch tx.TransactionType {
		case "income":
			months[month].Inflows = months[month].Inflows.Add(tx.Amount)
		case "expense":
			months[month].Outflows = months[month].Outflows.Add(tx.Amount)
		}
	}

	series := make([]models.MonthlyCashflow, 0, 12)
	for m := 1; m <= 12; m++ {
		months[m].Net = months[m].Inflows.Sub(months[m].Outflows)
		series = append(series, *months[m])
	}

	return &models.MonthlyCashflowResponse{
		Year:   year,
		Series: series,
	}, nil
}

func (s *ChartingService) GetCategoryUsageForYear(
	userID int64,
	year int,
	class string,
	accID *int64,
	catID *int64,
	asPercent bool,
) (*models.CategoryUsageResponse, error) {

	txs, err := s.TxRepo.GetTransactionsByYearAndClass(userID, year, class, accID)
	if err != nil {
		return nil, err
	}

	months := make(map[int]map[int64]decimal.Decimal)
	totals := make(map[int]decimal.Decimal)

	for m := 1; m <= 12; m++ {
		months[m] = make(map[int64]decimal.Decimal)
		totals[m] = decimal.NewFromInt(0)
	}

	for _, tx := range txs {

		if catID != nil {
			if tx.CategoryID == nil || *tx.CategoryID != *catID {
				continue
			}
		}

		month := int(tx.TxnDate.Month())
		var categoryID int64 = 0
		if tx.CategoryID != nil {
			categoryID = *tx.CategoryID
		}
		months[month][categoryID] = months[month][categoryID].Add(tx.Amount)
		totals[month] = totals[month].Add(tx.Amount)
	}

	var series []models.MonthlyCategoryUsage
	for m := 1; m <= 12; m++ {
		for catID, amt := range months[m] {
			entry := models.MonthlyCategoryUsage{
				Month:      m,
				CategoryID: catID,
				Category:   "",
				Amount:     amt,
			}
			if asPercent && !totals[m].IsZero() {
				perc := amt.Div(totals[m]).Mul(decimal.NewFromInt(100))
				entry.Percentage = &perc
			}
			series = append(series, entry)
		}
	}

	return &models.CategoryUsageResponse{
		Year:   year,
		Class:  class,
		Series: series,
	}, nil
}

func (s *ChartingService) GetCategoryUsageForYears(
	userID int64,
	years []int,
	class string,
	accID *int64,
	catID *int64,
	asPercent bool,
) (*models.MultiYearCategoryUsageResponse, error) {

	byYear := make(map[int]models.CategoryUsageResponse, len(years))
	yearStats := make(map[int]models.YearStat, len(years))

	var allYearsTotal decimal.Decimal = decimal.NewFromInt(0)
	var totalMonthsWithData int = 0

	for _, y := range years {
		one, err := s.GetCategoryUsageForYear(userID, y, class, accID, catID, asPercent)
		if err != nil {
			return nil, err
		}
		byYear[y] = *one

		var yearTotal decimal.Decimal = decimal.NewFromInt(0)
		monthsWithData := 0

		for _, entry := range one.Series {
			if entry.Amount.GreaterThan(decimal.NewFromInt(0)) {
				yearTotal = yearTotal.Add(entry.Amount)
				monthsWithData++
			}
		}

		var monthlyAvg decimal.Decimal = decimal.NewFromInt(0)
		if monthsWithData > 0 {
			monthlyAvg = yearTotal.Div(decimal.NewFromInt(int64(monthsWithData)))
		}

		yearStats[y] = models.YearStat{
			Total:          yearTotal,
			MonthlyAvg:     monthlyAvg,
			MonthsWithData: monthsWithData,
		}

		allYearsTotal = allYearsTotal.Add(yearTotal)
		totalMonthsWithData += monthsWithData
	}

	var allYearsAvg decimal.Decimal = decimal.NewFromInt(0)
	if totalMonthsWithData > 0 {
		allYearsAvg = allYearsTotal.Div(decimal.NewFromInt(int64(totalMonthsWithData)))
	}

	return &models.MultiYearCategoryUsageResponse{
		Years:  years,
		Class:  class,
		ByYear: byYear,
		Stats: models.MultiYearStats{
			YearStats:     yearStats,
			AllYearsTotal: allYearsTotal,
			AllYearsAvg:   allYearsAvg,
		},
	}, nil
}
