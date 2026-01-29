package services

import (
	"context"
	"sort"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

type StatisticsServiceInterface interface {
	GetAccountBasicStatistics(ctx context.Context, accID *int64, userID int64, year int) (*models.BasicAccountStats, error)
	GetAvailableStatsYears(ctx context.Context, accID *int64, userID int64) ([]int64, error)
	GetCurrentMonthStats(ctx context.Context, userID int64, accountID *int64) (*models.CurrentMonthStats, error)
	GetYearlyAverageForCategory(ctx context.Context, userID int64, accountID int64, categoryID int64, isGroup bool) (float64, error)
}

type StatisticsService struct {
	repo         *repositories.StatisticsRepository
	accRepo      *repositories.AccountRepository
	txnRepo      *repositories.TransactionRepository
	settingsRepo *repositories.SettingsRepository
}

func NewStatisticsService(
	repo *repositories.StatisticsRepository,
	accRepo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	settingsRepo *repositories.SettingsRepository,
) *StatisticsService {
	return &StatisticsService{
		repo:         repo,
		accRepo:      accRepo,
		txnRepo:      txnRepo,
		settingsRepo: settingsRepo,
	}
}

var _ StatisticsServiceInterface = (*StatisticsService)(nil)

func (s *StatisticsService) GetAccountBasicStatistics(ctx context.Context, accID *int64, userID int64, year int) (*models.BasicAccountStats, error) {

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

func (s *StatisticsService) GetAvailableStatsYears(ctx context.Context, accID *int64, userID int64) ([]int64, error) {
	return s.repo.GetAvailableStatsYears(ctx, nil, accID, userID)
}

func (s *StatisticsService) GetCurrentMonthStats(ctx context.Context, userID int64, accountID *int64) (*models.CurrentMonthStats, error) {

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

func (s *StatisticsService) GetTodayStats(ctx context.Context, userID int64, accountID *int64) (*models.TodayStats, error) {
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

func (s *StatisticsService) GetYearlyAverageForCategory(ctx context.Context, userID int64, accountID int64, categoryID int64, isGroup bool) (float64, error) {
	currentYear := time.Now().UTC().Year()

	if isGroup {
		return s.txnRepo.GetYearlyAverageForCategoryGroup(ctx, nil, userID, accountID, categoryID, currentYear)
	}

	return s.txnRepo.GetYearlyAverageForCategory(ctx, nil, userID, accountID, categoryID, currentYear)
}

func (s *StatisticsService) GetYearlyBreakdownStats(ctx context.Context, accID *int64, userID int64, year int, comparisonYear *int) (*models.YearlyBreakdownStats, error) {

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

func (s *StatisticsService) getYearStatsWithAllocations(ctx context.Context, accID *int64, userID int64, year int) (*models.YearStatsWithAllocations, error) {

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
