package queue_jobs_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type BackfillCashFlowsIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBackfillCashFlowsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BackfillCashFlowsIntegrationSuite))
}

type balanceRow struct {
	AsOf         time.Time
	StartBalance decimal.Decimal
	CashInflows  decimal.Decimal
	CashOutflows decimal.Decimal
	EndBalance   decimal.Decimal
}

type snapshotRow struct {
	AccountID  int64
	AsOf       time.Time
	EndBalance decimal.Decimal
}

func (s *BackfillCashFlowsIntegrationSuite) balances(accountID int64) []balanceRow {
	var rows []balanceRow
	err := s.TC.DB.WithContext(s.Ctx).
		Table("balances").
		Select("as_of, start_balance, cash_inflows, cash_outflows, end_balance").
		Where("account_id = ?", accountID).
		Order("as_of ASC").
		Scan(&rows).Error
	s.Require().NoError(err)
	return rows
}

func (s *BackfillCashFlowsIntegrationSuite) snapshots(userID int64) []snapshotRow {
	var rows []snapshotRow
	err := s.TC.DB.WithContext(s.Ctx).
		Table("account_daily_snapshots").
		Select("account_id, as_of, end_balance").
		Where("user_id = ?", userID).
		Order("account_id ASC, as_of ASC").
		Scan(&rows).Error
	s.Require().NoError(err)
	return rows
}

func (s *BackfillCashFlowsIntegrationSuite) newBackfillJob() *queue_jobs.BackfillAssetCashFlowsJob {
	return queue_jobs.NewBackfillAssetCashFlowsJob(
		zaptest.NewLogger(s.T()),
		queue_jobs.NewAdvisoryLock(s.TC.DB),
		s.TC.App.InvestmentService,
		2,
	)
}

func (s *BackfillCashFlowsIntegrationSuite) seedTradedAccount(userID int64, name string) int64 {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	opened := today.AddDate(0, 0, -10)
	initial := decimal.NewFromInt(100000)

	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          name,
		AccountTypeID: 5,
		Balance:       &initial,
		OpenedAt:      opened,
	})
	s.Require().NoError(err)

	assetID, err := s.TC.App.InvestmentService.InsertAsset(s.Ctx, userID, &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentETF,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
		Currency:       "EUR",
	})
	s.Require().NoError(err)

	fee := decimal.NewFromInt(3)
	_, err = s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TradeType:    models.InvestmentBuy,
		TxnDate:      opened.AddDate(0, 0, 2),
		Quantity:     decimal.NewFromInt(50),
		PricePerUnit: decimal.NewFromInt(100),
		Currency:     "EUR",
		Fee:          &fee,
	})
	s.Require().NoError(err)

	_, err = s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TradeType:    models.InvestmentSell,
		TxnDate:      opened.AddDate(0, 0, 5),
		Quantity:     decimal.NewFromInt(10),
		PricePerUnit: decimal.NewFromInt(110),
		Currency:     "EUR",
		Fee:          &fee,
	})
	s.Require().NoError(err)

	return accID
}

// AddToDailyBalance is additive, so a second run must land on the same numbers.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_RepeatRunIsIdempotent() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	job := s.newBackfillJob()

	s.Require().NoError(job.Process(s.Ctx))
	balancesAfterFirst := s.balances(accID)
	snapshotsAfterFirst := s.snapshots(userID)

	s.Require().NotEmpty(balancesAfterFirst, "first run should leave balance rows")
	s.Require().NotEmpty(snapshotsAfterFirst, "first run should leave snapshots")

	s.Require().NoError(job.Process(s.Ctx))

	s.Assert().Equal(balancesAfterFirst, s.balances(accID), "balances changed on the second run")
	s.Assert().Equal(snapshotsAfterFirst, s.snapshots(userID), "snapshots changed on the second run")
}

// Without the lock the two clear-then-add sequences interleave and every trade
// is counted twice.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_SkipsWhileLockHeld() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	job := s.newBackfillJob()
	s.Require().NoError(job.Process(s.Ctx))

	before := s.balances(accID)

	release, acquired, err := queue_jobs.NewAdvisoryLock(s.TC.DB).
		TryLock(s.Ctx, queue_jobs.LockKeyInvestmentRebuild)
	s.Require().NoError(err)
	s.Require().True(acquired)

	s.Require().NoError(job.Process(s.Ctx), "a held lock should not fail the job")
	s.Assert().Equal(before, s.balances(accID), "the skipped run touched the balances")

	release()

	s.Require().NoError(job.Process(s.Ctx))
	s.Assert().Equal(before, s.balances(accID))
}

// A closed account is never rebuilt, so the clear must not delete its history.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_KeepsClosedAccountSnapshots() {
	userID := int64(1)
	openAccID := s.seedTradedAccount(userID, "Brokerage")
	closedAccID := s.seedTradedAccount(userID, "Old Brokerage")

	job := s.newBackfillJob()
	s.Require().NoError(job.Process(s.Ctx))

	closedBefore := s.snapshots(userID)
	s.Require().NotEmpty(closedBefore)

	s.Require().NoError(s.TC.App.AccountService.CloseAccount(s.Ctx, userID, closedAccID))

	var countBefore int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Table("account_daily_snapshots").
		Where("account_id = ?", closedAccID).
		Count(&countBefore).Error)
	s.Require().Positive(countBefore, "closed account should still have snapshots before the run")

	s.Require().NoError(job.Process(s.Ctx))

	var countAfter int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Table("account_daily_snapshots").
		Where("account_id = ?", closedAccID).
		Count(&countAfter).Error)
	s.Assert().Equal(countBefore, countAfter, "the run deleted the closed account's snapshots")

	s.Assert().NotEmpty(s.balances(openAccID))
}

// An account with no balance rows must be skipped, not abort the whole rebuild.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_SkipsAccountWithoutBalances() {
	userID := int64(1)
	tradedAccID := s.seedTradedAccount(userID, "Brokerage")

	initial := decimal.NewFromInt(500)
	bareAccID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Bare Account",
		AccountTypeID: 1,
		Balance:       &initial,
		OpenedAt:      time.Now().UTC().Truncate(24 * time.Hour),
	})
	s.Require().NoError(err)

	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Exec("DELETE FROM balances WHERE account_id = ?", bareAccID).Error)

	s.Require().NoError(s.newBackfillJob().Process(s.Ctx))

	s.Assert().NotEmpty(s.balances(tradedAccID), "the traded account should still be rebuilt")

	// Only the snapshot rebuild reaches back to the opening day; the cash flow
	// step alone starts at the first trade.
	var earliest time.Time
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Table("account_daily_snapshots").
		Where("account_id = ?", tradedAccID).
		Select("MIN(as_of)").
		Scan(&earliest).Error)

	opening := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -10)
	s.Assert().Equal(opening, earliest.UTC(), "the rebuild stopped short for the traded account")
}

// A step that fails after the clear must roll the whole sequence back.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_FailureMidSequenceRollsBack() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	job := s.newBackfillJob()
	s.Require().NoError(job.Process(s.Ctx))

	balancesBefore := s.balances(accID)
	snapshotsBefore := s.snapshots(userID)
	s.Require().NotEmpty(balancesBefore)
	s.Require().NotEmpty(snapshotsBefore)

	// Reject snapshot writes, which happen after the clear.
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		`ALTER TABLE account_daily_snapshots ADD CONSTRAINT reject_all CHECK (false) NOT VALID`).Error)
	defer func() {
		s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
			`ALTER TABLE account_daily_snapshots DROP CONSTRAINT reject_all`).Error)
	}()

	s.Require().Error(job.Process(s.Ctx), "a failed user must fail the run so the queue retries it")

	s.Assert().Equal(balancesBefore, s.balances(accID), "the failed run changed the balances")
	s.Assert().Equal(snapshotsBefore, s.snapshots(userID), "the failed run changed the snapshots")
}
