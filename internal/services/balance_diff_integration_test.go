package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/suite"
)

type BalanceDiffIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBalanceDiffIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BalanceDiffIntegrationSuite))
}

func (s *BalanceDiffIntegrationSuite) seedAndDump(today time.Time) []tests.DailyBalanceRow {
	_, err := tests.SeedBalanceFixture(s.Ctx, s.TC.App, s.TC.DB, today)
	s.Require().NoError(err)

	rows, err := tests.DumpDailyBalances(s.Ctx, s.TC.DB)
	s.Require().NoError(err)
	return rows
}

// The gate for the balance schema migration. Every later phase must leave this
// dump untouched, so the fixture has to be repeatable first.
func (s *BalanceDiffIntegrationSuite) TestFixtureIsDeterministic() {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	first := s.seedAndDump(today)

	// 2 users x 3 accounts, each dense from the opening day (today-30) to today.
	s.Require().Len(first, 6*31, "the fixture did not seed every account for every day")

	// The credit card accounts must come out negative, or the liability sign path
	// is not in the diff at all.
	negative := 0
	for _, r := range first {
		if r.EndBalance.IsNegative() {
			negative++
		}
	}
	s.Require().Positive(negative, "no liability balances in the fixture")

	s.TruncateMutableTables()

	second := s.seedAndDump(today)

	s.Assert().Empty(tests.DiffDailyBalances(first, second), "the fixture is not deterministic")
}

// Guards the diff itself. A harness that cannot fail proves nothing.
func (s *BalanceDiffIntegrationSuite) TestDiffDetectsDrift() {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	first := s.seedAndDump(today)
	s.Require().NotEmpty(first)

	drifted := append([]tests.DailyBalanceRow(nil), first...)
	last := len(drifted) - 1
	drifted[last].EndBalance = drifted[last].EndBalance.Add(decimal.NewFromInt(1))

	s.Assert().NotEmpty(tests.DiffDailyBalances(first, drifted), "the diff missed a changed balance")
	s.Assert().NotEmpty(tests.DiffDailyBalances(first, first[:last]), "the diff missed a missing row")
}

// Phase 2's contract: the opening amount is a transaction like any other, so the
// ledger alone adds up to the balance the app reads.
func (s *BalanceDiffIntegrationSuite) TestOpeningBalanceIsATransaction() {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	s.seedAndDump(today)

	type row struct {
		AccountID int64
		Openings  int64
		TxnSum    decimal.Decimal
		EndBal    decimal.Decimal
	}

	var rows []row
	s.Require().NoError(s.TC.DB.Raw(`
		SELECT a.id AS account_id,
		       COUNT(*) FILTER (WHERE t.transaction_type = 'opening'
		                          AND t.category_id IS NOT NULL) AS openings,
		       COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0) AS txn_sum,
		       (SELECT b.end_balance FROM balances b
		         WHERE b.account_id = a.id ORDER BY b.as_of DESC LIMIT 1) AS end_bal
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id AND t.deleted_at IS NULL
		GROUP BY a.id
		ORDER BY a.id`).Scan(&rows).Error)

	s.Require().NotEmpty(rows)
	for _, r := range rows {
		s.Assert().EqualValues(1, r.Openings,
			"account %d needs exactly one opening transaction, with a category", r.AccountID)
		s.Assert().True(r.TxnSum.Round(4).Equal(r.EndBal.Round(4)),
			"account %d: transactions sum to %s, balance says %s",
			r.AccountID, r.TxnSum.StringFixed(4), r.EndBal.StringFixed(4))
	}
}
