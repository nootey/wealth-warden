package jobscheduler_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type AssetPriceHistoryBackfillJobTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAssetPriceHistoryBackfillJobSuite(t *testing.T) {
	suite.Run(t, new(AssetPriceHistoryBackfillJobTestSuite))
}

// Tests that the job runs without error when there are no assets
func (s *AssetPriceHistoryBackfillJobTestSuite) TestAssetPriceHistoryBackfillJob_NoAssets() {
	logger := zaptest.NewLogger(s.T())
	job := jobscheduler.NewAssetPriceHistoryBackfillJob(logger, s.TC.App.InvestmentService, s.TC.DB)

	err := job.Run(s.Ctx)
	s.NoError(err)
}

// Tests that the job backfills price history for an asset from its first trade date
func (s *AssetPriceHistoryBackfillJobTestSuite) TestAssetPriceHistoryBackfillJob_BackfillsFromFirstTrade() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	threeDaysAgo := today.AddDate(0, 0, -3)

	initialBalance := decimal.NewFromInt(100000)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      threeDaysAgo,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Insert a trade 3 days ago — this sets the earliest trade date
	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      threeDaysAgo,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Clear any price history that was inserted by InsertInvestmentTrade
	// so we can verify the backfill job inserts it
	err = s.TC.DB.WithContext(s.Ctx).
		Exec("DELETE FROM asset_price_history WHERE asset_id = ?", assetID).Error
	s.Require().NoError(err)

	// Verify no price history exists yet
	var countBefore int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.AssetPriceHistory{}).
		Where("asset_id = ?", assetID).
		Count(&countBefore).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), countBefore, "no price history should exist before backfill")

	// Run the backfill job
	logger := zaptest.NewLogger(s.T())
	job := jobscheduler.NewAssetPriceHistoryBackfillJob(logger, s.TC.App.InvestmentService, s.TC.DB)

	ctx3, cancel3 := context.WithTimeout(s.Ctx, 60*time.Second)
	defer cancel3()

	err = job.Run(ctx3)
	if err != nil {
		if errors.Is(ctx3.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: backfill job timed out")
		}
		s.Require().NoError(err)
	}

	// Verify price history was created
	var countAfter int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.AssetPriceHistory{}).
		Where("asset_id = ?", assetID).
		Count(&countAfter).Error
	s.Require().NoError(err)
	s.Assert().Greater(countAfter, int64(0), "price history should be populated after backfill")

	// Verify all prices are valid
	var priceHistory []models.AssetPriceHistory
	err = s.TC.DB.WithContext(s.Ctx).
		Where("asset_id = ?", assetID).
		Order("as_of ASC").
		Find(&priceHistory).Error
	s.Require().NoError(err)

	for _, ph := range priceHistory {
		s.Assert().True(ph.Price.GreaterThan(decimal.Zero),
			"price on %s should be > 0, got %s", ph.AsOf.Format("2006-01-02"), ph.Price.String())
		// Verify no weekend entries
		s.Assert().NotEqual(time.Saturday, ph.AsOf.Weekday(),
			"should not have price history for Saturday")
		s.Assert().NotEqual(time.Sunday, ph.AsOf.Weekday(),
			"should not have price history for Sunday")
	}
}

// Tests that the job is idempotent — running it twice doesn't duplicate records
func (s *AssetPriceHistoryBackfillJobTestSuite) TestAssetPriceHistoryBackfillJob_Idempotent() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "EUR",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	logger := zaptest.NewLogger(s.T())
	job := jobscheduler.NewAssetPriceHistoryBackfillJob(logger, s.TC.App.InvestmentService, s.TC.DB)

	// Run once
	ctx3, cancel3 := context.WithTimeout(s.Ctx, 30*time.Second)
	defer cancel3()

	err = job.Run(ctx3)
	if err != nil {
		if errors.Is(ctx3.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: backfill job timed out")
		}
		s.Require().NoError(err)
	}

	var countAfterFirst int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.AssetPriceHistory{}).
		Where("asset_id = ?", assetID).
		Count(&countAfterFirst).Error
	s.Require().NoError(err)

	// Run again
	ctx4, cancel4 := context.WithTimeout(s.Ctx, 30*time.Second)
	defer cancel4()

	err = job.Run(ctx4)
	if err != nil {
		if errors.Is(ctx4.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: backfill job timed out")
		}
		s.Require().NoError(err)
	}

	var countAfterSecond int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.AssetPriceHistory{}).
		Where("asset_id = ?", assetID).
		Count(&countAfterSecond).Error
	s.Require().NoError(err)

	s.Assert().Equal(countAfterFirst, countAfterSecond,
		"running backfill twice should not create duplicate records")
}
