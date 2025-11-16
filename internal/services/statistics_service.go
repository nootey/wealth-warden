package services

import (
	"sort"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"

	"github.com/shopspring/decimal"
)

type StatisticsService struct {
	Config  *config.Config
	Ctx     *DefaultServiceContext
	Repo    *repositories.StatisticsRepository
	AccRepo *repositories.AccountRepository
	TxRepo  *repositories.TransactionRepository
}

func NewStatisticsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.StatisticsRepository,
	accRepo *repositories.AccountRepository,
	txRepo *repositories.TransactionRepository,
) *StatisticsService {
	return &StatisticsService{
		Ctx:     ctx,
		Config:  cfg,
		Repo:    repo,
		AccRepo: accRepo,
		TxRepo:  txRepo,
	}
}

func (s *StatisticsService) GetAccountBasicStatistics(accID *int64, userID int64, year int) (*models.BasicAccountStats, error) {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if accID != nil {
		if _, err := s.AccRepo.FindAccountByID(tx, *accID, userID, true); err != nil {
			return nil, err
		}
	}

	tot, err := s.Repo.FetchYearlyTotals(tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow, _ := decimal.NewFromString(tot.InflowText)
	outflow, _ := decimal.NewFromString(tot.OutflowText)
	net, _ := decimal.NewFromString(tot.NetText)

	mrows, err := s.Repo.FetchMonthlyTotals(tx, userID, accID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	takeHomeYear := decimal.Zero
	overflowYear := decimal.Zero
	activeMonthsForAvg := 0

	for _, mr := range mrows {
		inm, _ := decimal.NewFromString(mr.InflowText)
		outm, _ := decimal.NewFromString(mr.OutflowText)
		netm := inm.Add(outm)

		if !inm.IsZero() || !outm.IsZero() {
			activeMonthsForAvg++
		}

		if netm.GreaterThan(decimal.Zero) {
			takeHomeYear = takeHomeYear.Add(netm)
		} else if netm.LessThan(decimal.Zero) {
			overflowYear = overflowYear.Add(netm)
		}
	}

	takeHome := takeHomeYear
	overflow := overflowYear

	avgTakeHome := decimal.Zero
	avgOverflow := decimal.Zero
	if activeMonthsForAvg > 0 {
		div := decimal.NewFromInt(int64(activeMonthsForAvg))
		if !takeHome.IsZero() {
			avgTakeHome = takeHome.Div(div)
		}
		if !overflow.IsZero() {
			avgOverflow = overflow.Div(div)
		}
	}

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

	rows, err := s.Repo.FetchYearlyCategoryTotals(tx, userID, accID, year)
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

func (s *StatisticsService) GetAvailableStatsYears(accID *int64, userID int64) ([]int64, error) {
	return s.Repo.GetAvailableStatsYears(accID, userID)
}

func (s *StatisticsService) GetCurrentMonthStats(userID int64, accountID *int64) (*models.CurrentMonthStats, error) {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
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
	var err error
	var checkingAccounts []models.Account

	if accountID != nil {
		acc, errAcc := s.AccRepo.FindAccountByID(tx, *accountID, userID, false)
		if errAcc != nil {
			tx.Rollback()
			return nil, errAcc
		}
		checkingAccounts = []models.Account{*acc}
		mrows, err = s.Repo.FetchMonthlyTotals(tx, userID, accountID, year)
	} else {
		checkingAccounts, err = s.AccRepo.FindAccountsBySubtype(tx, userID, "checking", true)
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
		mrows, err = s.Repo.FetchMonthlyTotalsCheckingOnly(tx, userID, accountIDs, year)
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

	transfers, err := s.TxRepo.GetMonthlyTransfersFromChecking(tx, userID, accountIDs, year, month)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	savingsTotal := decimal.Zero
	investmentsTotal := decimal.Zero

	for _, tr := range transfers {

		if tr.TransactionInflow.Account.AccountType.Subtype == "savings" {
			savingsTotal = savingsTotal.Add(tr.Amount)
		}
		if tr.TransactionInflow.Account.AccountType.Type == "investment" {
			investmentsTotal = investmentsTotal.Add(tr.Amount)
		}

		takeHome = takeHome.Sub(tr.Amount)
	}

	savingsRate := decimal.Zero
	investRate := decimal.Zero
	if !inflow.IsZero() {
		savingsRate = savingsTotal.Div(inflow)
		investRate = investmentsTotal.Div(inflow)
	}

	if takeHome.LessThan(decimal.Zero) {
		overflow = takeHome
		takeHome = decimal.Zero
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &models.CurrentMonthStats{
		UserID:      userID,
		AccountID:   accountID,
		Currency:    models.DefaultCurrency,
		Year:        year,
		Month:       month,
		Inflow:      inflow,
		Outflow:     outflow,
		Net:         net,
		TakeHome:    takeHome,
		Overflow:    overflow,
		Savings:     savingsTotal,
		Investments: investmentsTotal,
		SavingsRate: savingsRate,
		InvestRate:  investRate,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (s *StatisticsService) GetYearlyAverageForCategory(userID int64, accountID int64, categoryID int64, isGroup bool) (float64, error) {
	currentYear := time.Now().Year()

	if isGroup {
		return s.TxRepo.GetYearlyAverageForCategoryGroup(userID, accountID, categoryID, currentYear)
	}

	return s.TxRepo.GetYearlyAverageForCategory(userID, accountID, categoryID, currentYear)
}
