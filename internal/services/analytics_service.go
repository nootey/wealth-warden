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
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

type AnalyticsServiceInterface interface {
	GetNetWorthSeries(ctx context.Context, userID int64, currency, rangeKey, from, to string, accountID *int64) (*models.NetWorthResponse, error)
	GetCategoryUsageForYear(ctx context.Context, userID int64, year int, class string, accID, catID *int64, asPercent bool) (*models.CategoryUsageResponse, error)
	GetCategoryUsageForYears(ctx context.Context, userID int64, years []int, class string, accID, catID *int64, asPercent bool) (*models.MultiYearCategoryUsageResponse, error)
	GetYearlyCashFlowBreakdown(ctx context.Context, userID int64, year int, accountID *int64) (*models.YearlyCashflowBreakdown, error)
	GetYearlySankeyData(ctx context.Context, userID int64, accountID *int64, year int) (*models.YearlySankeyData, error)
	GetAccountBasicStatistics(ctx context.Context, accID *int64, userID int64, year int) (*models.BasicAccountStats, error)
	GetAvailableStatsYears(ctx context.Context, accID *int64, userID int64) ([]int64, error)
	GetCurrentMonthStats(ctx context.Context, userID int64, accountID *int64) (*models.CurrentMonthStats, error)
	GetYearlyAverageForCategory(ctx context.Context, userID int64, accountID int64, categoryID int64, isGroup bool) (float64, error)
}
type AnalyticsService struct {
	repo         repositories.AnalyticsRepositoryInterface
	accRepo      repositories.AccountRepositoryInterface
	txnRepo      repositories.TransactionRepositoryInterface
	settingsRepo repositories.SettingsRepositoryInterface
}

func NewAnalyticsService(
	repo *repositories.AnalyticsRepository,
	accRepo *repositories.AccountRepository,
	txRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
) *AnalyticsService {
	return &AnalyticsService{
		repo:         repo,
		accRepo:      accRepo,
		txnRepo:      txRepo,
		settingsRepo: settingsRepo,
	}
}

var _ AnalyticsServiceInterface = (*AnalyticsService)(nil)

func (s *AnalyticsService) GetNetWorthSeries(ctx context.Context, userID int64, currency, rangeKey, from, to string, accountID *int64) (*models.NetWorthResponse, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	var dfrom, dto time.Time

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

	points, err := s.repo.FetchNetWorthSeries(ctx, tx, userID, currency, dfrom, dto, gran, accountID)
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
	curDate, curStr, err := s.repo.FetchLatestNetWorth(ctx, tx, userID, currency, accountID)
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

	var at *models.AccountType
	if accountID != nil {
		at, err = s.accRepo.FindAccountTypeByAccID(ctx, tx, *accountID, userID)
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

	if at != nil {
		nwRes.AssetType = &at.Classification
	}

	return nwRes, nil
}

func (s *AnalyticsService) GetYearlyCashFlowBreakdown(ctx context.Context, userID int64, year int, accountID *int64) (*models.YearlyCashflowBreakdown, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Validate account if provided
	if accountID != nil {
		if _, err := s.accRepo.FindAccountByID(ctx, tx, *accountID, userID, true); err != nil {
			return nil, err
		}
	}

	// Get monthly totals
	mrows, err := s.repo.FetchMonthlyTotals(ctx, tx, userID, accountID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var shouldSubtractTransfers bool
	var transferAccountIDs []int64

	if accountID != nil {
		acc, err := s.accRepo.FindAccountByID(ctx, tx, *accountID, userID, false)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if acc.AccountType.Subtype == "checking" {
			shouldSubtractTransfers = true
			transferAccountIDs = []int64{*accountID}
		}
	} else {
		checkingAccounts, err := s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(checkingAccounts) > 0 {
			shouldSubtractTransfers = true
			for _, a := range checkingAccounts {
				transferAccountIDs = append(transferAccountIDs, a.ID)
			}
		}
	}

	// Fetch all transfers for the year at once
	var allTransfers []models.Transfer
	if shouldSubtractTransfers {
		allTransfers, err = s.txnRepo.GetYearlyTransfersFromChecking(ctx, tx, userID, transferAccountIDs, year)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Group by month
	transfersByMonth := make(map[int][]models.Transfer)
	for _, tr := range allTransfers {
		month := int(tr.CreatedAt.Month())
		transfersByMonth[month] = append(transfersByMonth[month], tr)
	}

	months := make([]models.MonthBreakdown, 0, 12)

	for month := 1; month <= 12; month++ {
		var inflow, outflow, investments, savings, debtRepayments decimal.Decimal

		for _, mr := range mrows {
			if mr.Month == month {
				inflow, _ = decimal.NewFromString(mr.InflowText)
				outflow, _ = decimal.NewFromString(mr.OutflowText)
				break
			}
		}

		net := inflow.Add(outflow)

		// Get transfers and calculate categories
		if shouldSubtractTransfers {

			for _, tr := range transfersByMonth[month] {

				isSavings, isInvestment, isDebt := utils.CategorizeTransferDestination(&tr.TransactionInflow.Account.AccountType)

				if isSavings {
					savings = savings.Add(tr.Amount)
				} else if isInvestment {
					investments = investments.Add(tr.Amount)
				} else if isDebt {
					debtRepayments = debtRepayments.Add(tr.Amount)
				}

			}
		}

		categorizedTransfersTotal := savings.Add(investments).Add(debtRepayments)
		takeHomeCalc := net.Sub(categorizedTransfersTotal)

		var takeHome, overflow decimal.Decimal
		if takeHomeCalc.LessThan(decimal.Zero) {
			overflow = takeHomeCalc
		} else {
			takeHome = takeHomeCalc
		}

		months = append(months, models.MonthBreakdown{
			Month: month,
			Categories: models.MonthCategories{
				Inflows:        inflow,
				Outflows:       outflow,
				Investments:    investments,
				Savings:        savings,
				DebtRepayments: debtRepayments,
				TakeHome:       takeHome,
				Overflow:       overflow,
			},
		})
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.YearlyCashflowBreakdown{
		Year:   year,
		Months: months,
	}, nil
}

func (s *AnalyticsService) GetCategoryUsageForYear(ctx context.Context, userID int64, year int, class string, accID, catID *int64, asPercent bool) (*models.CategoryUsageResponse, error) {

	txs, err := s.txnRepo.GetTransactionsByYearAndClass(ctx, nil, userID, year, class, accID)
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

func (s *AnalyticsService) GetCategoryUsageForYears(ctx context.Context, userID int64, years []int, class string, accID, catID *int64, asPercent bool) (*models.MultiYearCategoryUsageResponse, error) {

	type yearResult struct {
		year int
		data *models.CategoryUsageResponse
		err  error
	}

	resultsChan := make(chan yearResult, len(years))

	// Fetch all years concurrently
	for _, y := range years {
		go func(year int) {
			data, err := s.GetCategoryUsageForYear(ctx, userID, year, class, accID, catID, asPercent)
			resultsChan <- yearResult{year: year, data: data, err: err}
		}(y)
	}

	// Collect results
	byYear := make(map[int]models.CategoryUsageResponse, len(years))
	yearStats := make(map[int]models.YearStat, len(years))

	for range years {
		result := <-resultsChan
		if result.err != nil {
			return nil, result.err
		}

		byYear[result.year] = *result.data

		var yearTotal = decimal.NewFromInt(0)
		monthsWithData := make(map[int]bool)

		for _, entry := range result.data.Series {
			if entry.Amount.GreaterThan(decimal.NewFromInt(0)) {
				yearTotal = yearTotal.Add(entry.Amount)
				monthsWithData[entry.Month] = true
			}
		}

		var monthlyAvg = decimal.NewFromInt(0)
		if len(monthsWithData) > 0 {
			monthlyAvg = yearTotal.Div(decimal.NewFromInt(int64(len(monthsWithData))))
		}

		yearStats[result.year] = models.YearStat{
			Total:          yearTotal,
			MonthlyAvg:     monthlyAvg,
			MonthsWithData: len(monthsWithData),
		}
	}

	// Get all-time stats from the database
	allTimeTotal, allTimeMonths, err := s.txnRepo.GetAllTimeStatsByClass(ctx, nil, userID, class, accID, catID)
	if err != nil {
		return nil, err
	}

	var allTimeAvg = decimal.NewFromInt(0)
	if allTimeMonths > 0 {
		allTimeAvg = allTimeTotal.Div(decimal.NewFromInt(int64(allTimeMonths)))
	}

	return &models.MultiYearCategoryUsageResponse{
		Years:  years,
		Class:  class,
		ByYear: byYear,
		Stats: models.MultiYearYCategoryStats{
			YearStats:     yearStats,
			AllTimeTotal:  allTimeTotal,
			AllTimeAvg:    allTimeAvg,
			AllTimeMonths: allTimeMonths,
		},
	}, nil
}

func (s *AnalyticsService) GetYearlySankeyData(ctx context.Context, userID int64, accountID *int64, year int) (*models.YearlySankeyData, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var checkingAccounts []models.Account
	var accountIDs []int64

	if accountID != nil {
		// Validate account belongs to user
		if _, err := s.accRepo.FindAccountByID(ctx, tx, *accountID, userID, false); err != nil {
			return nil, err
		}

		accountIDs = []int64{*accountID}
	} else {
		checkingAccounts, err = s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			return nil, err
		}
		if len(checkingAccounts) == 0 {
			tx.Commit()
			return nil, nil
		}

		accountIDs = make([]int64, len(checkingAccounts))
		for i, a := range checkingAccounts {
			accountIDs[i] = a.ID
		}
	}

	yearlyTotals, err := s.repo.FetchYearlyTotals(ctx, tx, userID, accountID, year)
	if err != nil {
		return nil, err
	}
	totalIncome, _ := decimal.NewFromString(yearlyTotals.InflowText)

	// Get all transfers for the year
	savings := decimal.Zero
	investments := decimal.Zero
	debtRepayments := decimal.Zero

	transfers, err := s.txnRepo.GetYearlyTransfersFromChecking(ctx, tx, userID, accountIDs, year)
	if err != nil {
		return nil, err
	}

	for _, tr := range transfers {
		isSavings, isInvestment, isDebt := utils.CategorizeTransferDestination(&tr.TransactionInflow.Account.AccountType)
		if isSavings {
			savings = savings.Add(tr.Amount)
		}
		if isInvestment {
			investments = investments.Add(tr.Amount)
		}
		if isDebt {
			debtRepayments = debtRepayments.Add(tr.Amount)
		}
	}

	// Get expense categories
	categoryRows, err := s.repo.FetchYearlyCategoryTotals(ctx, tx, userID, accountID, year)
	if err != nil {
		return nil, err
	}

	expenseCategories := []models.CategoryFlow{}
	totalExpenses := decimal.Zero

	for _, cat := range categoryRows {
		outflow, _ := decimal.NewFromString(cat.OutflowText)
		totalExpenses = totalExpenses.Add(outflow)

		categoryName := "Uncategorized"
		if cat.DisplayName != nil {
			categoryName = *cat.DisplayName
		}

		expenseCategories = append(expenseCategories, models.CategoryFlow{
			CategoryID:   cat.CategoryID,
			CategoryName: categoryName,
			Amount:       outflow,
			Percentage:   decimal.Zero,
		})
	}

	// Calculate percentages
	for i := range expenseCategories {
		if !totalExpenses.IsZero() {
			expenseCategories[i].Percentage = expenseCategories[i].Amount.Div(totalExpenses).Mul(decimal.NewFromInt(100))
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.YearlySankeyData{
		Year:              year,
		Currency:          models.DefaultCurrency,
		TotalIncome:       totalIncome,
		Savings:           savings,
		Investments:       investments,
		DebtRepayments:    debtRepayments,
		Expenses:          totalExpenses,
		ExpenseCategories: expenseCategories,
	}, nil
}

func (s *AnalyticsService) GetAccountBasicStatistics(ctx context.Context, accID *int64, userID int64, year int) (*models.BasicAccountStats, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if accID != nil {
		if _, err := s.accRepo.FindAccountByID(ctx, tx, *accID, userID, true); err != nil {
			return nil, err
		}
	}

	tot, err := s.repo.FetchYearlyTotals(ctx, tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow, _ := decimal.NewFromString(tot.InflowText)
	outflow, _ := decimal.NewFromString(tot.OutflowText)
	net, _ := decimal.NewFromString(tot.NetText)

	mrows, err := s.repo.FetchMonthlyTotals(ctx, tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	takeHomeYear := decimal.Zero
	overflowYear := decimal.Zero
	takeHomeMonthCount := 0
	overflowMonthCount := 0

	var shouldSubtractTransfers bool
	var transferAccountIDs []int64

	if accID != nil {
		acc, errAcc := s.accRepo.FindAccountByID(ctx, tx, *accID, userID, false)
		if errAcc != nil {
			tx.Rollback()
			return nil, errAcc
		}
		if acc.AccountType.Subtype == "checking" {
			shouldSubtractTransfers = true
			transferAccountIDs = []int64{*accID}
		}
	} else {
		checkingAccounts, err := s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(checkingAccounts) > 0 {
			shouldSubtractTransfers = true
			transferAccountIDs = make([]int64, len(checkingAccounts))
			for i, a := range checkingAccounts {
				transferAccountIDs[i] = a.ID
			}
		}
	}

	// Fetch all transfers for the year at once
	var allTransfers []models.Transfer
	if shouldSubtractTransfers {
		allTransfers, err = s.txnRepo.GetYearlyTransfersFromChecking(ctx, tx, userID, transferAccountIDs, year)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Group transfers by month
	transfersByMonth := make(map[int][]models.Transfer)
	for _, tr := range allTransfers {
		month := int(tr.TransactionOutflow.TxnDate.Month())
		transfersByMonth[month] = append(transfersByMonth[month], tr)
	}

	// Process each month using pre-grouped transfers
	for _, mr := range mrows {
		inm, _ := decimal.NewFromString(mr.InflowText)
		outm, _ := decimal.NewFromString(mr.OutflowText)
		netm := inm.Add(outm)

		if shouldSubtractTransfers {

			categorizedTransfersTotal := decimal.Zero

			for _, tr := range transfersByMonth[mr.Month] {
				isSavings, isInvestment, isDebt := utils.CategorizeTransferDestination(&tr.TransactionInflow.Account.AccountType)

				if isSavings || isInvestment || isDebt {
					categorizedTransfersTotal = categorizedTransfersTotal.Add(tr.Amount)
				}
			}

			netm = netm.Sub(categorizedTransfersTotal)
		}

		if netm.GreaterThan(decimal.Zero) {
			takeHomeYear = takeHomeYear.Add(netm)
			takeHomeMonthCount++
		} else if netm.LessThan(decimal.Zero) {
			overflowYear = overflowYear.Add(netm)
			overflowMonthCount++
		}
	}

	takeHome := takeHomeYear
	overflow := overflowYear

	avgTakeHome := takeHome.Div(decimal.NewFromInt(12))
	avgOverflow := overflow.Div(decimal.NewFromInt(12))

	activeMonths := tot.ActiveMonths
	if activeMonths < 1 {
		activeMonths = 0
	}

	avgIn := decimal.Zero
	avgOut := decimal.Zero
	if activeMonths > 0 {
		avgIn = inflow.Div(decimal.NewFromInt(int64(activeMonths)))
		avgOut = outflow.Div(decimal.NewFromInt(int64(activeMonths)))
	}

	rows, err := s.repo.FetchYearlyCategoryTotals(ctx, tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var cats []models.CategoryStat
	for _, r := range rows {
		ci, _ := decimal.NewFromString(r.InflowText)
		co, _ := decimal.NewFromString(r.OutflowText)
		cn, _ := decimal.NewFromString(r.NetText)

		var pctIn, pctOut float64
		if !inflow.IsZero() {
			pctIn = ci.Div(inflow).InexactFloat64() * 100.0
		}
		if !outflow.IsZero() {
			pctOut = co.Div(outflow).InexactFloat64() * 100.0
		}

		cats = append(cats, models.CategoryStat{
			CategoryID:   r.CategoryID,
			CategoryName: r.DisplayName,
			Inflow:       ci,
			Outflow:      co,
			Net:          cn,
			PctOfInflow:  pctIn,
			PctOfOutflow: pctOut,
		})
	}

	sort.Slice(cats, func(i, j int) bool {
		return cats[i].Outflow.GreaterThan(cats[j].Outflow)
	})

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.BasicAccountStats{
		UserID:             userID,
		AccountID:          accID,
		Currency:           models.DefaultCurrency,
		Year:               year,
		Inflow:             inflow,
		Outflow:            outflow,
		Net:                net,
		AvgMonthlyInflow:   avgIn,
		AvgMonthlyOutflow:  avgOut,
		TakeHome:           takeHome,
		Overflow:           overflow,
		AvgMonthlyTakeHome: avgTakeHome,
		AvgMonthlyOverflow: avgOverflow,
		ActiveMonths:       activeMonths,
		Categories:         cats,
		GeneratedAt:        time.Now().UTC(),
	}, nil
}

func (s *AnalyticsService) GetAvailableStatsYears(ctx context.Context, accID *int64, userID int64) ([]int64, error) {
	return s.repo.GetAvailableStatsYears(ctx, nil, accID, userID)
}

func (s *AnalyticsService) GetCurrentMonthStats(ctx context.Context, userID int64, accountID *int64) (*models.CurrentMonthStats, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())

	var mrows []models.MonthlyTotalsRow
	var checkingAccounts []models.Account

	if accountID != nil {
		acc, errAcc := s.accRepo.FindAccountByID(ctx, tx, *accountID, userID, false)
		if errAcc != nil {
			tx.Rollback()
			return nil, errAcc
		}
		checkingAccounts = []models.Account{*acc}
		mrows, err = s.repo.FetchMonthlyTotals(ctx, tx, userID, accountID, year)
	} else {
		checkingAccounts, err = s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(checkingAccounts) == 0 {
			tx.Commit()
			return nil, nil
		}

		accountIDs := make([]int64, len(checkingAccounts))
		for i, a := range checkingAccounts {
			accountIDs[i] = a.ID
		}
		mrows, err = s.repo.FetchMonthlyTotalsCheckingOnly(ctx, tx, userID, accountIDs, year)
	}

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow := decimal.Zero
	outflow := decimal.Zero
	net := decimal.Zero
	for _, mr := range mrows {
		if mr.Month == month {
			inflow, _ = decimal.NewFromString(mr.InflowText)
			outflow, _ = decimal.NewFromString(mr.OutflowText)
			net, _ = decimal.NewFromString(mr.NetText)
			break
		}
	}

	takeHome := net
	overflow := decimal.Zero

	accountIDs := make([]int64, len(checkingAccounts))
	for i, a := range checkingAccounts {
		accountIDs[i] = a.ID
	}

	transfers, err := s.txnRepo.GetMonthlyTransfersFromChecking(ctx, tx, userID, accountIDs, year, month)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	savingsTotal := decimal.Zero
	investmentsTotal := decimal.Zero
	debtRepaymentTotal := decimal.Zero

	for _, tr := range transfers {

		isSavings, isInvestment, isDebt := utils.CategorizeTransferDestination(&tr.TransactionInflow.Account.AccountType)

		if isSavings {
			savingsTotal = savingsTotal.Add(tr.Amount)
		}
		if isInvestment {
			investmentsTotal = investmentsTotal.Add(tr.Amount)
		}
		if isDebt {
			debtRepaymentTotal = debtRepaymentTotal.Add(tr.Amount)
		}

	}

	savingsRate := decimal.Zero
	investRate := decimal.Zero
	repaymentRate := decimal.Zero
	if !inflow.IsZero() {
		savingsRate = savingsTotal.Div(inflow)
		investRate = investmentsTotal.Div(inflow)
		repaymentRate = debtRepaymentTotal.Div(inflow)
	}

	// Calculate total categorized transfers
	categorizedTransfersTotal := savingsTotal.Add(investmentsTotal).Add(debtRepaymentTotal)

	takeHome = net.Sub(categorizedTransfersTotal)

	if takeHome.LessThan(decimal.Zero) {
		overflow = takeHome
		takeHome = decimal.Zero
	}

	// Get expense categories for current month
	var categoryRows []models.YearlyCategoryRow
	if accountID != nil {
		categoryRows, err = s.repo.FetchMonthlyCategoryTotals(ctx, tx, userID, accountID, year, month)
	} else {
		categoryRows, err = s.repo.FetchMonthlyCategoryTotalsCheckingOnly(ctx, tx, userID, accountIDs, year, month)
	}
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var cats []models.CategoryStat
	for _, cat := range categoryRows {
		outflow, _ := decimal.NewFromString(cat.OutflowText)

		// Only include expenses
		if outflow.LessThan(decimal.Zero) {
			cats = append(cats, models.CategoryStat{
				CategoryID:   cat.CategoryID,
				CategoryName: cat.DisplayName,
				Outflow:      outflow.Abs(),
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.CurrentMonthStats{
		UserID:            userID,
		AccountID:         accountID,
		Currency:          models.DefaultCurrency,
		Year:              year,
		Month:             month,
		Inflow:            inflow,
		Outflow:           outflow,
		Net:               net,
		TakeHome:          takeHome,
		Overflow:          overflow,
		Savings:           savingsTotal,
		Investments:       investmentsTotal,
		DebtRepayments:    debtRepaymentTotal,
		SavingsRate:       savingsRate,
		InvestRate:        investRate,
		DebtRepaymentRate: repaymentRate,
		GeneratedAt:       time.Now().UTC(),
		Categories:        cats,
	}, nil
}

func (s *AnalyticsService) GetTodayStats(ctx context.Context, userID int64, accountID *int64) (*models.TodayStats, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	today := utils.LocalMidnightUTC(time.Now(), loc)

	var row *models.MonthlyTotalsRow
	var checkingAccounts []models.Account

	if accountID != nil {
		// Verify acc is valid
		acc, errAcc := s.accRepo.FindAccountByID(ctx, tx, *accountID, userID, false)
		if errAcc != nil {
			tx.Rollback()
			return nil, errAcc
		}
		row, err = s.repo.FetchDailyTotals(ctx, tx, userID, &acc.ID, today)
	} else {
		checkingAccounts, err = s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(checkingAccounts) == 0 {
			tx.Commit()
			return nil, nil
		}

		accountIDs := make([]int64, len(checkingAccounts))
		for i, a := range checkingAccounts {
			accountIDs[i] = a.ID
		}
		row, err = s.repo.FetchDailyTotalsCheckingOnly(ctx, tx, userID, accountIDs, today)
	}

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow, _ := decimal.NewFromString(row.InflowText)
	outflow, _ := decimal.NewFromString(row.OutflowText)
	net, _ := decimal.NewFromString(row.NetText)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.TodayStats{
		UserID:      userID,
		AccountID:   accountID,
		Currency:    models.DefaultCurrency,
		Inflow:      inflow,
		Outflow:     outflow,
		Net:         net,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (s *AnalyticsService) GetYearlyAverageForCategory(ctx context.Context, userID int64, accountID int64, categoryID int64, isGroup bool) (float64, error) {
	currentYear := time.Now().UTC().Year()

	if isGroup {
		return s.txnRepo.GetYearlyAverageForCategoryGroup(ctx, nil, userID, accountID, categoryID, currentYear)
	}

	return s.txnRepo.GetYearlyAverageForCategory(ctx, nil, userID, accountID, categoryID, currentYear)
}

func (s *AnalyticsService) GetYearlyBreakdownStats(ctx context.Context, accID *int64, userID int64, year int, comparisonYear *int) (*models.YearlyBreakdownStats, error) {

	currentStats, err := s.getYearStatsWithAllocations(ctx, accID, userID, year)
	if err != nil {
		return nil, err
	}

	compareYear := year - 1
	if comparisonYear != nil {
		compareYear = *comparisonYear
	}

	var comparisonStats *models.YearStatsWithAllocations
	if compareYear > 0 {
		stats, err := s.getYearStatsWithAllocations(ctx, accID, userID, compareYear)
		if err == nil {
			comparisonStats = stats
		}
	}

	return &models.YearlyBreakdownStats{
		CurrentYear:    currentStats,
		ComparisonYear: comparisonStats,
	}, nil
}

func (s *AnalyticsService) getYearStatsWithAllocations(ctx context.Context, accID *int64, userID int64, year int) (*models.YearStatsWithAllocations, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if accID != nil {
		if _, err := s.accRepo.FindAccountByID(ctx, tx, *accID, userID, true); err != nil {
			return nil, err
		}
	}

	tot, err := s.repo.FetchYearlyTotals(ctx, tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow, _ := decimal.NewFromString(tot.InflowText)
	outflow, _ := decimal.NewFromString(tot.OutflowText)

	mrows, err := s.repo.FetchMonthlyTotals(ctx, tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	takeHomeYear := decimal.Zero
	overflowYear := decimal.Zero

	savingsAllocated := decimal.Zero
	investmentAllocated := decimal.Zero
	debtAllocated := decimal.Zero

	var shouldSubtractTransfers bool
	var transferAccountIDs []int64

	if accID != nil {
		acc, errAcc := s.accRepo.FindAccountByID(ctx, tx, *accID, userID, false)
		if errAcc != nil {
			tx.Rollback()
			return nil, errAcc
		}
		if acc.AccountType.Subtype == "checking" {
			shouldSubtractTransfers = true
			transferAccountIDs = []int64{*accID}
		}
	} else {
		checkingAccounts, err := s.accRepo.FindAccountsBySubtype(ctx, tx, userID, "checking", true)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(checkingAccounts) > 0 {
			shouldSubtractTransfers = true
			transferAccountIDs = make([]int64, len(checkingAccounts))
			for i, a := range checkingAccounts {
				transferAccountIDs[i] = a.ID
			}
		}
	}

	var allTransfers []models.Transfer
	if shouldSubtractTransfers {
		allTransfers, err = s.txnRepo.GetYearlyTransfersFromChecking(ctx, tx, userID, transferAccountIDs, year)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	transfersByMonth := make(map[int][]models.Transfer)
	for _, tr := range allTransfers {
		month := int(tr.TransactionOutflow.TxnDate.Month())
		transfersByMonth[month] = append(transfersByMonth[month], tr)
	}

	for _, mr := range mrows {
		inm, _ := decimal.NewFromString(mr.InflowText)
		outm, _ := decimal.NewFromString(mr.OutflowText)
		netm := inm.Add(outm)

		if shouldSubtractTransfers {
			categorizedTransfersTotal := decimal.Zero

			for _, tr := range transfersByMonth[mr.Month] {
				isSavings, isInvestment, isDebt := utils.CategorizeTransferDestination(&tr.TransactionInflow.Account.AccountType)

				if isSavings {
					savingsAllocated = savingsAllocated.Add(tr.Amount)
					categorizedTransfersTotal = categorizedTransfersTotal.Add(tr.Amount)
				}
				if isInvestment {
					investmentAllocated = investmentAllocated.Add(tr.Amount)
					categorizedTransfersTotal = categorizedTransfersTotal.Add(tr.Amount)
				}
				if isDebt {
					debtAllocated = debtAllocated.Add(tr.Amount)
					categorizedTransfersTotal = categorizedTransfersTotal.Add(tr.Amount)
				}
			}

			netm = netm.Sub(categorizedTransfersTotal)
		}

		if netm.GreaterThan(decimal.Zero) {
			takeHomeYear = takeHomeYear.Add(netm)
		} else if netm.LessThan(decimal.Zero) {
			overflowYear = overflowYear.Add(netm)
		}
	}

	totalAllocated := savingsAllocated.Add(investmentAllocated).Add(debtAllocated)

	var savingsPct, investmentPct, debtPct float64
	if !inflow.IsZero() {
		savingsPct = savingsAllocated.Div(inflow).InexactFloat64() * 100.0
		investmentPct = investmentAllocated.Div(inflow).InexactFloat64() * 100.0
		debtPct = debtAllocated.Div(inflow).InexactFloat64() * 100.0
	}

	avgTakeHome := takeHomeYear.Div(decimal.NewFromInt(12))
	avgOverflow := overflowYear.Div(decimal.NewFromInt(12))

	activeMonths := tot.ActiveMonths
	if activeMonths < 1 {
		activeMonths = 0
	}

	avgIn := decimal.Zero
	avgOut := decimal.Zero
	if activeMonths > 0 {
		avgIn = inflow.Div(decimal.NewFromInt(int64(activeMonths)))
		avgOut = outflow.Div(decimal.NewFromInt(int64(activeMonths)))
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.YearStatsWithAllocations{
		Year:                year,
		Inflow:              inflow,
		Outflow:             outflow,
		AvgMonthlyInflow:    avgIn,
		AvgMonthlyOutflow:   avgOut,
		TakeHome:            takeHomeYear,
		Overflow:            overflowYear,
		AvgMonthlyTakeHome:  avgTakeHome,
		AvgMonthlyOverflow:  avgOverflow,
		SavingsAllocated:    savingsAllocated,
		InvestmentAllocated: investmentAllocated,
		DebtAllocated:       debtAllocated,
		TotalAllocated:      totalAllocated,
		SavingsPct:          savingsPct,
		InvestmentPct:       investmentPct,
		DebtPct:             debtPct,
	}, nil
}
