package jobs_test

import (
	"context"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/tests"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
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

type backfillRunner struct {
	worker *jobs.BackfillAssetCashFlowsWorker
}

func (r backfillRunner) Run(ctx context.Context) error {
	return r.worker.Work(ctx, &river.Job[jobqueue.BackfillAssetCashFlowsArgs]{JobRow: &rivertype.JobRow{}})
}

func (s *BackfillCashFlowsIntegrationSuite) newBackfillJob() backfillRunner {
	return backfillRunner{worker: jobs.NewBackfillAssetCashFlowsWorker(
		zaptest.NewLogger(s.T()),
		s.TC.App.InvestmentService,
		2,
	)}
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

	s.Require().NoError(job.Run(s.Ctx))
	balancesAfterFirst := s.balances(accID)
	snapshotsAfterFirst := s.snapshots(userID)

	s.Require().NotEmpty(balancesAfterFirst, "first run should leave balance rows")
	s.Require().NotEmpty(snapshotsAfterFirst, "first run should leave snapshots")

	s.Require().NoError(job.Run(s.Ctx))

	s.Assert().Equal(balancesAfterFirst, s.balances(accID), "balances changed on the second run")
	s.Assert().Equal(snapshotsAfterFirst, s.snapshots(userID), "snapshots changed on the second run")
}

// A closed account is never rebuilt, so the clear must not delete its history.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_KeepsClosedAccountSnapshots() {
	userID := int64(1)
	openAccID := s.seedTradedAccount(userID, "Brokerage")
	closedAccID := s.seedTradedAccount(userID, "Old Brokerage")

	job := s.newBackfillJob()
	s.Require().NoError(job.Run(s.Ctx))

	closedBefore := s.snapshots(userID)
	s.Require().NotEmpty(closedBefore)

	// Closed directly: the close flow needs an empty account, and this test only
	// needs the closed flag.
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Exec("UPDATE accounts SET is_active = false, closed_at = NOW() WHERE id = ?", closedAccID).Error)

	var countBefore int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Table("account_daily_snapshots").
		Where("account_id = ?", closedAccID).
		Count(&countBefore).Error)
	s.Require().Positive(countBefore, "closed account should still have snapshots before the run")

	s.Require().NoError(job.Run(s.Ctx))

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

	s.Require().NoError(s.newBackfillJob().Run(s.Ctx))

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
	s.Require().NoError(job.Run(s.Ctx))

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

	s.Require().Error(job.Run(s.Ctx), "a failed user must fail the run so the queue retries it")

	s.Assert().Equal(balancesBefore, s.balances(accID), "the failed run changed the balances")
	s.Assert().Equal(snapshotsBefore, s.snapshots(userID), "the failed run changed the snapshots")
}

// Criterion 2: before trades became system transactions, a transactions-only rebuild
// wiped their cash, and import_service re-applied every trade by hand to hide it.
func (s *BackfillCashFlowsIntegrationSuite) TestRebuildFromTransactions_KeepsTradeCash() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	before := s.balances(accID)
	s.Require().NotEmpty(before)

	// Guard against a vacuous pass: the trades must have moved cash at all.
	last := before[len(before)-1]
	s.Require().False(last.EndBalance.Equal(decimal.NewFromInt(100000)),
		"the fixture trades did not move any cash")

	opening := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -10)
	repo := repositories.NewAccountRepository(s.TC.DB)
	s.Require().NoError(repo.RebuildFromTransactions(s.Ctx, nil, userID, accID, "EUR", opening))

	s.Assert().Equal(before, s.balances(accID), "the rebuild erased trade cash")

	s.Require().NoError(repo.RebuildFromTransactions(s.Ctx, nil, userID, accID, "EUR", opening))
	s.Assert().Equal(before, s.balances(accID), "the second rebuild drifted")
}

// The migration path: legacy trades carry no transaction, and the backfill has to
// create one for each without moving a single balance.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_LinksLegacyTrades() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	before := s.balances(accID)
	s.Require().NotEmpty(before)

	s.stripTradeLinks(userID)

	s.Require().NoError(s.newBackfillJob().Run(s.Ctx))

	s.Assert().Equal(before, s.balances(accID), "the backfill changed the balances")

	var unlinked int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Table("investment_trades").
		Where("user_id = ? AND transaction_id IS NULL", userID).
		Count(&unlinked).Error)
	s.Assert().Zero(unlinked, "the backfill left trades without a transaction")
}

// Puts the user's trades back into their pre-phase-1 shape: cash written straight
// into balances, no transaction behind it.
func (s *BackfillCashFlowsIntegrationSuite) stripTradeLinks(userID int64) {
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(`
		DELETE FROM transactions
		WHERE id IN (SELECT transaction_id FROM investment_trades
		             WHERE user_id = ? AND transaction_id IS NOT NULL)`, userID).Error)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		"UPDATE investment_trades SET transaction_id = NULL WHERE user_id = ?", userID).Error)
}

// The insert trigger blocks posting to a closed account, so the backfill suspends it.
func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_LinksLegacyTradesOnClosedAccount() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Old Brokerage")

	before := s.balances(accID)
	s.Require().NotEmpty(before)

	s.stripTradeLinks(userID)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Exec("UPDATE accounts SET is_active = false, closed_at = NOW() WHERE id = ?", accID).Error)

	s.Require().NoError(s.newBackfillJob().Run(s.Ctx), "the backfill failed on a closed account")

	s.Assert().Equal(before, s.balances(accID), "the backfill changed the balances")

	// The trigger has to be back on, or every later write to a closed account passes.
	err := s.TC.DB.WithContext(s.Ctx).Exec(`
		INSERT INTO transactions (user_id, account_id, direction, amount, currency, txn_date, transaction_type)
		VALUES (?, ?, 'expense', 1, 'EUR', NOW(), 'ledger')`, userID, accID).Error
	s.Assert().Error(err, "the closed-account trigger was left disabled")
}

func (s *BackfillCashFlowsIntegrationSuite) TestBackfill_RelinksSoftDeletedTradeTransaction() {
	userID := int64(1)
	accID := s.seedTradedAccount(userID, "Brokerage")

	before := s.balances(accID)
	s.Require().NotEmpty(before)

	var staleID int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Raw(`
		SELECT it.transaction_id
		FROM   investment_trades it
		JOIN   investment_assets ia ON ia.id = it.asset_id
		WHERE  ia.account_id = ? AND it.trade_type = 'buy'`, accID).Scan(&staleID).Error)
	s.Require().NotZero(staleID, "the fixture buy has no cash transaction")

	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Exec(
		"UPDATE transactions SET deleted_at = NOW() WHERE id = ?", staleID).Error)

	s.Require().NoError(s.newBackfillJob().Run(s.Ctx))

	s.Assert().Equal(before, s.balances(accID), "the backfill left the buy without cash")

	var unbacked int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Raw(`
		SELECT count(*)
		FROM   investment_trades it
		LEFT   JOIN transactions t ON t.id = it.transaction_id AND t.deleted_at IS NULL
		WHERE  it.user_id = ? AND t.id IS NULL`, userID).Scan(&unbacked).Error)
	s.Assert().Zero(unbacked, "a trade is still without a live transaction")
}
