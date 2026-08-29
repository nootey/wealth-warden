package services_test

import (
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"

	"github.com/shopspring/decimal"
)

// A bounded recompute (non-nil from) must rewrite only the snapshots on or after
// that date and leave the earlier ones as they were.
func (s *InvestmentServiceTestSuite) TestUpdateSnapshotMarketValues_FromDateBoundsTheRecompute() {
	repo := repositories.NewAccountRepository(s.TC.DB)
	asset := s.createAssetRow("MVBOUND")
	userID := int64(1)

	day := func(n int) time.Time {
		return time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, n)
	}

	s.Require().NoError(s.TC.DB.Create(&models.InvestmentTrade{
		UserID:       userID,
		AssetID:      asset.ID,
		TxnDate:      day(-10),
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(2),
		PricePerUnit: decimal.NewFromInt(100),
		ValueAtBuy:   decimal.NewFromInt(200),
		Currency:     "EUR",
	}).Error)

	s.Require().NoError(s.TC.DB.Create(&models.TickerPriceHistory{
		Ticker:   asset.Ticker,
		AsOf:     day(-10),
		Price:    decimal.NewFromInt(50),
		Currency: "EUR",
	}).Error)

	sentinel := decimal.NewFromInt(999)
	for n := -5; n <= 0; n++ {
		s.Require().NoError(s.TC.DB.Create(&models.AccountDailySnapshot{
			UserID:      userID,
			AccountID:   asset.AccountID,
			AsOf:        day(n),
			EndBalance:  decimal.Zero,
			MarketValue: sentinel,
			Currency:    "EUR",
		}).Error)
	}

	from := day(-2)
	s.Require().NoError(repo.UpdateSnapshotMarketValues(s.Ctx, nil, userID, &from))

	marketValueOn := func(n int) decimal.Decimal {
		var snap models.AccountDailySnapshot
		s.Require().NoError(
			s.TC.DB.Where("account_id = ? AND as_of = ?", asset.AccountID, day(n)).First(&snap).Error,
		)
		return snap.MarketValue
	}

	for _, n := range []int{-5, -4, -3} {
		s.Truef(sentinel.Equal(marketValueOn(n)), "day %d changed, got %s", n, marketValueOn(n))
	}

	// price 50 * quantity 2
	want := decimal.NewFromInt(100)
	for _, n := range []int{-2, -1, 0} {
		s.Truef(want.Equal(marketValueOn(n)), "day %d not recomputed, got %s", n, marketValueOn(n))
	}
}
