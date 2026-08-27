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

type cancellingFetcher struct {
	tests.MockPriceFetcher
	calls  int
	limit  int
	cancel context.CancelFunc
}

func (f *cancellingFetcher) GetAssetPriceOnDate(_ context.Context, ticker string, _ models.InvestmentType, _ time.Time) (*finance.PriceData, error) {
	f.calls++
	if f.calls >= f.limit {
		f.cancel()
	}
	return &finance.PriceData{Symbol: ticker, Price: 100.0, Currency: "EUR", LastUpdate: 1700000000}, nil
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

func (s *InvestmentServiceTestSuite) countPriceHistory(assetID int64) int64 {
	var count int64
	s.Require().NoError(
		s.TC.DB.Model(&models.AssetPriceHistory{}).Where("asset_id = ?", assetID).Count(&count).Error,
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

func (s *InvestmentServiceTestSuite) TestBackfillAssetPriceHistory_KeepsProgressOnCancel() {
	asset := s.createAssetRow("IWDA.AS")

	ctx, cancel := context.WithCancel(s.Ctx)
	defer cancel()
	fetcher := &cancellingFetcher{limit: 105, cancel: cancel}
	svc := s.newInvestmentService(fetcher)

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -250)

	err := svc.BackfillAssetPriceHistory(ctx, asset.ID, asset.Ticker, asset.InvestmentType, from, to)

	s.Require().ErrorIs(err, context.Canceled)
	s.Equal(int64(100), s.countPriceHistory(asset.ID))
}

func (s *InvestmentServiceTestSuite) TestBackfillAssetPriceHistory_AllFetchesFailReturnsError() {
	asset := s.createAssetRow("NOSUCH.AS")
	svc := s.newInvestmentService(&tests.MockPriceFetcher{})

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -10)

	err := svc.BackfillAssetPriceHistory(s.Ctx, asset.ID, asset.Ticker, asset.InvestmentType, from, to)

	s.Require().Error(err)
	s.Contains(err.Error(), "NOSUCH.AS")
	s.Zero(s.countPriceHistory(asset.ID))
}

func (s *InvestmentServiceTestSuite) TestBackfillAssetPriceHistory_WritesKnownTicker() {
	asset := s.createAssetRow("IWDA.AS")
	svc := s.newInvestmentService(&tests.MockPriceFetcher{})

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -10)

	s.Require().NoError(
		svc.BackfillAssetPriceHistory(s.Ctx, asset.ID, asset.Ticker, asset.InvestmentType, from, to),
	)
	s.Positive(s.countPriceHistory(asset.ID))
}
