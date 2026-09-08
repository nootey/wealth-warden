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

type AccountBalanceIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAccountBalanceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AccountBalanceIntegrationSuite))
}

func (s *AccountBalanceIntegrationSuite) seedFixture() []int64 {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := tests.SeedBalanceFixture(s.Ctx, s.TC.App, s.TC.DB, today)
	s.Require().NoError(err)

	var ids []int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM accounts ORDER BY id`).Scan(&ids).Error)
	s.Require().NotEmpty(ids)
	return ids
}

// The migration fills account_balances with the same statement RecomputeFromTransactions
// runs. Phase 4's contract is that the number it produces is the one the old chain ends on.
func (s *AccountBalanceIntegrationSuite) TestRecomputeMatchesTheOldChain() {
	ids := s.seedFixture()
	repo := repositories.NewBalanceRepository(s.TC.DB)

	for _, id := range ids {
		s.Require().NoError(repo.RecomputeFromTransactions(s.Ctx, nil, id))
	}

	type row struct {
		AccountID int64
		Balance   decimal.Decimal
		EndBal    decimal.Decimal
	}
	var rows []row
	s.Require().NoError(s.TC.DB.Raw(`
		SELECT ab.account_id,
		       ab.balance,
		       (SELECT b.end_balance FROM balances b
		         WHERE b.account_id = ab.account_id ORDER BY b.as_of DESC LIMIT 1) AS end_bal
		FROM account_balances ab
		ORDER BY ab.account_id`).Scan(&rows).Error)

	s.Require().Len(rows, len(ids), "every account needs a balance row")
	for _, r := range rows {
		s.Assert().True(r.Balance.Round(4).Equal(r.EndBal.Round(4)),
			"account %d: account_balances says %s, the chain ends on %s",
			r.AccountID, r.Balance.StringFixed(4), r.EndBal.StringFixed(4))
	}
}

func (s *AccountBalanceIntegrationSuite) TestApplyDeltaAccumulates() {
	ids := s.seedFixture()
	repo := repositories.NewBalanceRepository(s.TC.DB)

	id := ids[0]
	s.Require().NoError(repo.RecomputeFromTransactions(s.Ctx, nil, id))

	before, err := repo.GetBalance(s.Ctx, nil, id)
	s.Require().NoError(err)

	s.Require().NoError(repo.ApplyDelta(s.Ctx, nil, id, decimal.NewFromInt(100)))
	s.Require().NoError(repo.ApplyDelta(s.Ctx, nil, id, decimal.NewFromInt(-30)))

	after, err := repo.GetBalance(s.Ctx, nil, id)
	s.Require().NoError(err)
	s.Assert().True(after.Sub(before).Equal(decimal.NewFromInt(70)),
		"two deltas moved the balance by %s, want 70", after.Sub(before).StringFixed(4))

	// A recompute reads only transactions, so it must undo deltas nothing backed.
	s.Require().NoError(repo.RecomputeFromTransactions(s.Ctx, nil, id))
	healed, err := repo.GetBalance(s.Ctx, nil, id)
	s.Require().NoError(err)
	s.Assert().True(healed.Equal(before), "recompute left %s, want %s",
		healed.StringFixed(4), before.StringFixed(4))
}

// ApplyDelta seeds its own row, so an account that never had one is not a special case.
func (s *AccountBalanceIntegrationSuite) TestApplyDeltaSeedsAMissingRow() {
	ids := s.seedFixture()
	repo := repositories.NewBalanceRepository(s.TC.DB)

	id := ids[0]
	s.Require().NoError(s.TC.DB.Exec(`DELETE FROM account_balances WHERE account_id = ?`, id).Error)

	zero, err := repo.GetBalance(s.Ctx, nil, id)
	s.Require().NoError(err)
	s.Require().True(zero.IsZero(), "a missing row must read as zero, got %s", zero.StringFixed(4))

	s.Require().NoError(repo.ApplyDelta(s.Ctx, nil, id, decimal.NewFromInt(25)))

	got, err := repo.GetBalance(s.Ctx, nil, id)
	s.Require().NoError(err)
	s.Assert().True(got.Equal(decimal.NewFromInt(25)), "got %s, want 25", got.StringFixed(4))
}

// The dual write contract: after any write path, the one row balance equals the sum
// of the account's live transactions, with no rebuild in between.
func (s *AccountBalanceIntegrationSuite) assertBalanceMatchesLedger(context string) {
	type row struct {
		AccountID int64
		Balance   decimal.Decimal
		TxnSum    decimal.Decimal
	}
	var rows []row
	s.Require().NoError(s.TC.DB.Raw(`
		SELECT a.id AS account_id,
		       COALESCE(ab.balance, 0) AS balance,
		       COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0) AS txn_sum
		FROM accounts a
		LEFT JOIN account_balances ab ON ab.account_id = a.id
		LEFT JOIN transactions t ON t.account_id = a.id AND t.deleted_at IS NULL
		GROUP BY a.id, ab.balance
		ORDER BY a.id`).Scan(&rows).Error)

	s.Require().NotEmpty(rows)
	for _, r := range rows {
		s.Assert().True(r.Balance.Round(4).Equal(r.TxnSum.Round(4)),
			"%s: account %d holds %s, its transactions sum to %s",
			context, r.AccountID, r.Balance.StringFixed(4), r.TxnSum.StringFixed(4))
	}
}

func (s *AccountBalanceIntegrationSuite) fixtureAccountID(name string) int64 {
	var id int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM accounts WHERE name = ?`, name).Scan(&id).Error)
	s.Require().NotZero(id, "fixture account %q is missing", name)
	return id
}

func (s *AccountBalanceIntegrationSuite) TestWritePathsKeepTheOneRowBalance() {
	userID := int64(1)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	s.seedFixture()

	// The seeders go through the same seams, so the fixture alone must already agree.
	s.assertBalanceMatchesLedger("after the fixture seed")

	checkingID := s.fixtureAccountID("Fixture Checking A")
	brokerID := s.fixtureAccountID("Fixture Brokerage A")
	cardID := s.fixtureAccountID("Fixture Card A")

	s.Run("transaction insert", func() {
		amount := decimal.NewFromInt(64)
		_, err := s.TC.App.TransactionService.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
			AccountID: checkingID,
			Direction: "expense",
			Amount:    amount,
			TxnDate:   today.AddDate(0, 0, -3),
		})
		s.Require().NoError(err)
		s.assertBalanceMatchesLedger("after a transaction insert")
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
			Amount:     txn.Amount.Add(decimal.NewFromInt(17)),
			TxnDate:    txn.TxnDate,
		})
		s.Require().NoError(err)
		s.assertBalanceMatchesLedger("after a transaction edit")
	})

	s.Run("transaction delete", func() {
		var id int64
		s.Require().NoError(s.TC.DB.Raw(`
			SELECT id FROM transactions
			WHERE account_id = ? AND transaction_type = 'ledger' AND deleted_at IS NULL
			ORDER BY id LIMIT 1`, checkingID).Scan(&id).Error)
		s.Require().NotZero(id)

		s.Require().NoError(s.TC.App.TransactionService.DeleteTransaction(s.Ctx, userID, id))
		s.assertBalanceMatchesLedger("after a transaction delete")
	})

	s.Run("transfer", func() {
		_, err := s.TC.App.TransactionService.InsertTransfer(s.Ctx, userID, &models.TransferReq{
			SourceID:      checkingID,
			DestinationID: cardID,
			Amount:        decimal.NewFromInt(50),
			CreatedAt:     today.AddDate(0, 0, -5),
		})
		s.Require().NoError(err)
		s.assertBalanceMatchesLedger("after a transfer")
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
		s.assertBalanceMatchesLedger("after a manual balance edit")
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
		s.assertBalanceMatchesLedger("after a trade delete")
	})
}
