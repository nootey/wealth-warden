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
}

func NewStatisticsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.StatisticsRepository,
	accRepo *repositories.AccountRepository,
) *StatisticsService {
	return &StatisticsService{
		Ctx:     ctx,
		Config:  cfg,
		Repo:    repo,
		AccRepo: accRepo,
	}
}

func (s *StatisticsService) GetAccountBasicStatistics(accID, userID int64, year int) (*models.BasicAccountStats, error) {

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

	acc, err := s.AccRepo.FindAccountByID(tx, accID, userID, true)
	if err != nil {
		return nil, err
	}

	tot, err := s.Repo.FetchYearlyTotals(tx, userID, &acc.ID, year)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	inflow, _ := decimal.NewFromString(tot.InflowText)
	outflow, _ := decimal.NewFromString(tot.OutflowText)
	net, _ := decimal.NewFromString(tot.NetText)

	mrows, err := s.Repo.FetchMonthlyTotals(tx, userID, &acc.ID, year)
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
			overflowYear = overflowYear.Add(netm.Neg())
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

	rows, err := s.Repo.FetchYearlyCategoryTotals(tx, userID, &acc.ID, year)
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
		AccountID:          &acc.ID,
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
