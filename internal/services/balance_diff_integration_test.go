package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
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

	s.assertLedgerMatchesBalances("after the fixture seed")
}

func (s *BalanceDiffIntegrationSuite) assertLedgerMatchesBalances(context string) {
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
			"%s: account %d needs exactly one opening transaction, with a category", context, r.AccountID)
		s.Assert().True(r.TxnSum.Round(4).Equal(r.EndBal.Round(4)),
			"%s: account %d: transactions sum to %s, balance says %s",
			context, r.AccountID, r.TxnSum.StringFixed(4), r.EndBal.StringFixed(4))
	}
}

func (s *BalanceDiffIntegrationSuite) accountID(name string) int64 {
	var id int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM accounts WHERE name = ?`, name).Scan(&id).Error)
	s.Require().NotZero(id, "fixture account %q is missing", name)
	return id
}

func (s *BalanceDiffIntegrationSuite) TestEveryWritePathLeavesATransaction() {
	userID := int64(1)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	s.seedAndDump(today)

	checkingID := s.accountID("Fixture Checking A")
	brokerID := s.accountID("Fixture Brokerage A")
	cardID := s.accountID("Fixture Card A")

	s.Run("transfer", func() {
		_, err := s.TC.App.TransactionService.InsertTransfer(s.Ctx, userID, &models.TransferReq{
			SourceID:      checkingID,
			DestinationID: cardID,
			Amount:        decimal.NewFromInt(50),
			CreatedAt:     today.AddDate(0, 0, -5),
		})
		s.Require().NoError(err)
		s.assertLedgerMatchesBalances("after a transfer")
	})

	s.Run("dividend", func() {
		var assetID int64
		s.Require().NoError(s.TC.DB.Raw(
			`SELECT id FROM investment_assets WHERE account_id = ?`, brokerID).Scan(&assetID).Error)
		s.Require().NotZero(assetID)

		amount := decimal.NewFromInt(30)
		tax := decimal.NewFromInt(5)
		_, err := s.TC.App.InvestmentService.CreateInvestmentIncome(s.Ctx, userID, &models.InvestmentIncomeReq{
			AssetID:     assetID,
			TxnDate:     today.AddDate(0, 0, -6),
			IncomeType:  models.IncomeTypeDividend,
			Amount:      &amount,
			TaxWithheld: &tax,
			Currency:    "EUR",
		})
		s.Require().NoError(err)
		s.assertLedgerMatchesBalances("after dividend income")
	})

	s.Run("balance edit", func() {
		var typeID int64
		s.Require().NoError(s.TC.DB.Raw(
			`SELECT account_type_id FROM accounts WHERE id = ?`, checkingID).Scan(&typeID).Error)

		desired := decimal.NewFromInt(777)
		_, err := s.TC.App.AccountService.UpdateAccount(s.Ctx, userID, checkingID, &models.AccountReq{
			Name:          "Fixture Checking A",
			AccountTypeID: typeID,
			Balance:       &desired,
		})
		s.Require().NoError(err)
		s.assertLedgerMatchesBalances("after a manual balance edit")
	})

	s.Run("transaction update", func() {
		var txn models.Transaction
		s.Require().NoError(s.TC.DB.Raw(`
			SELECT * FROM transactions
			WHERE account_id = ? AND transaction_type = 'ledger' AND deleted_at IS NULL
			ORDER BY id LIMIT 1`, checkingID).Scan(&txn).Error)
		s.Require().NotZero(txn.ID)

		_, err := s.TC.App.TransactionService.UpdateTransaction(s.Ctx, userID, txn.ID, &models.TransactionReq{
			AccountID:  txn.AccountID,
			CategoryID: txn.CategoryID,
			Direction:  txn.Direction,
			Amount:     txn.Amount.Add(decimal.NewFromInt(11)),
			TxnDate:    txn.TxnDate,
		})
		s.Require().NoError(err)
		s.assertLedgerMatchesBalances("after a transaction edit")
	})

	s.Run("transaction delete", func() {
		var id int64
		s.Require().NoError(s.TC.DB.Raw(`
			SELECT id FROM transactions
			WHERE account_id = ? AND transaction_type = 'ledger' AND deleted_at IS NULL
			ORDER BY id LIMIT 1`, checkingID).Scan(&id).Error)
		s.Require().NotZero(id)

		s.Require().NoError(s.TC.App.TransactionService.DeleteTransaction(s.Ctx, userID, id))
		s.assertLedgerMatchesBalances("after a transaction delete")
	})

	s.Run("trade delete", func() {
		var id int64
		s.Require().NoError(s.TC.DB.Raw(`
			SELECT it.id FROM investment_trades it
			JOIN investment_assets ia ON ia.id = it.asset_id
			WHERE ia.account_id = ? AND it.trade_type = 'sell'
			ORDER BY it.id LIMIT 1`, brokerID).Scan(&id).Error)
		s.Require().NotZero(id)

		s.Require().NoError(s.TC.App.InvestmentService.DeleteInvestmentTrade(s.Ctx, userID, id))
		s.assertLedgerMatchesBalances("after a trade delete")
	})
}

func (s *BalanceDiffIntegrationSuite) TestDailyTableMatchesTheBalanceChain() {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	fromTransactions := s.seedAndDump(today)
	s.Require().NotEmpty(fromTransactions)

	type acct struct {
		ID       int64
		UserID   int64
		Currency string
		OpenedAt time.Time
	}
	var accounts []acct
	s.Require().NoError(s.TC.DB.Raw(
		`SELECT id, user_id, currency, opened_at FROM accounts ORDER BY id`).Scan(&accounts).Error)
	s.Require().NotEmpty(accounts)

	s.Require().NoError(s.TC.DB.Exec(`TRUNCATE TABLE account_daily_snapshots`).Error)

	repo := repositories.NewAccountRepository(s.TC.DB)
	for _, a := range accounts {
		s.Require().NoError(repo.UpsertSnapshotsFromBalances(
			s.Ctx, nil, a.UserID, a.ID, a.Currency,
			a.OpenedAt.UTC().Truncate(24*time.Hour), today))
	}

	fromBalances, err := tests.DumpDailyBalances(s.Ctx, s.TC.DB)
	s.Require().NoError(err)

	s.Assert().Empty(tests.DiffDailyBalances(fromBalances, fromTransactions))
}
