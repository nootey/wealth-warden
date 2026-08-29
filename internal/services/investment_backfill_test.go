package services_test

import (
	"context"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap/zaptest"
)

type closedMarketFetcher struct {
	tests.MockPriceFetcher
}

func (f *closedMarketFetcher) GetAssetPriceRange(context.Context, string, time.Time, time.Time) ([]finance.DatedPrice, error) {
	return nil, nil
}

func (s *InvestmentServiceTestSuite) createAssetRow(ticker string) models.InvestmentAsset {
	acc := models.Account{
		UserID:            1,
		Name:              s.T().Name(),
		AccountTypeID:     5,
		Currency:          "EUR",
		BalanceProjection: "fixed",
		ExpectedBalance:   decimal.Zero,
		OpenedAt:          time.Now().UTC(),
		IsActive:          true,
	}
	s.Require().NoError(s.TC.DB.Create(&acc).Error)

	asset := models.InvestmentAsset{
		AccountID:       acc.ID,
		UserID:          1,
		InvestmentType:  models.InvestmentStock,
		Name:            ticker,
		Ticker:          ticker,
		Quantity:        decimal.NewFromInt(1),
		AverageBuyPrice: decimal.NewFromInt(100),
		ValueAtBuy:      decimal.NewFromInt(100),
		CurrentValue:    decimal.NewFromInt(100),
		Currency:        "EUR",
	}
	s.Require().NoError(s.TC.DB.Create(&asset).Error)
	return asset
}

func (s *InvestmentServiceTestSuite) countPriceHistory(ticker string) int64 {
	var count int64
	s.Require().NoError(
		s.TC.DB.Model(&models.TickerPriceHistory{}).Where("ticker = ?", ticker).Count(&count).Error,
	)
	return count
}

func (s *InvestmentServiceTestSuite) newInvestmentService(fetcher finance.PriceFetcher) *services.InvestmentService {
	return services.NewInvestmentService(
		zaptest.NewLogger(s.T()),
		repositories.NewInvestmentRepository(s.TC.DB),
		repositories.NewAccountRepository(s.TC.DB),
		repositories.NewTransactionRepository(s.TC.DB),
		repositories.NewSettingsRepository(s.TC.DB),
		jobqueue.NoopDispatcher{},
		fetcher,
	)
}

func (s *InvestmentServiceTestSuite) TestBackfillTickerPriceHistory_ClosedMarketIsNotAnError() {
	asset := s.createAssetRow("IWDA.AS")
	svc := s.newInvestmentService(&closedMarketFetcher{})

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -10)

	s.Require().NoError(
		svc.BackfillTickerPriceHistory(s.Ctx, asset.Ticker, from, to),
	)
	s.Zero(s.countPriceHistory(asset.Ticker))
}

func (s *InvestmentServiceTestSuite) TestBackfillTickerPriceHistory_FetchFailureReturnsError() {
	asset := s.createAssetRow("NOSUCH.AS")
	svc := s.newInvestmentService(&tests.MockPriceFetcher{})

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -10)

	err := svc.BackfillTickerPriceHistory(s.Ctx, asset.Ticker, from, to)

	s.Require().Error(err)
	s.Contains(err.Error(), "NOSUCH.AS")
	s.Zero(s.countPriceHistory(asset.Ticker))
}

func (s *InvestmentServiceTestSuite) TestBackfillTickerPriceHistory_WritesKnownTicker() {
	asset := s.createAssetRow("IWDA.AS")
	svc := s.newInvestmentService(&tests.MockPriceFetcher{})

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -10)

	s.Require().NoError(
		svc.BackfillTickerPriceHistory(s.Ctx, asset.Ticker, from, to),
	)
	s.Positive(s.countPriceHistory(asset.Ticker))

	written := s.countPriceHistory(asset.Ticker)
	s.Require().NoError(
		svc.BackfillTickerPriceHistory(s.Ctx, asset.Ticker, from, to),
	)
	s.Equal(written, s.countPriceHistory(asset.Ticker))
}
