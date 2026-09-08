package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type BalanceServiceSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBalanceServiceSuite(t *testing.T) {
	suite.Run(t, new(BalanceServiceSuite))
}

// The root user from the basic seed.
const seedUserID int64 = 1

// Account type ids follow the insert order in 010_seed_account_types.go.
const (
	checkingTypeID   = 1  // cash / checking
	brokerageTypeID  = 5  // investment / brokerage
	creditCardTypeID = 18 // credit_card / credit, classified as a liability
)

func (s *BalanceServiceSuite) insertAccount(name string, typeID, opening int64) int64 {
	balance := decimal.NewFromInt(opening)
	id, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          name,
		AccountTypeID: typeID,
		Balance:       &balance,
		OpenedAt:      s.today().AddDate(0, 0, -30),
	})
	s.Require().NoError(err)
	return id
}

func (s *BalanceServiceSuite) today() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

// A checking account, a brokerage holding one asset traded twice, and a credit
// card. The card is here because InsertAccount negates a liability's opening
// amount, and that sign has to survive every write path below.
func (s *BalanceServiceSuite) seedAccounts() []int64 {
	today := s.today()

	checkingID := s.insertAccount("Checking", checkingTypeID, 1000)
	brokerID := s.insertAccount("Brokerage", brokerageTypeID, 5000)
	cardID := s.insertAccount("Card", creditCardTypeID, 500)

	txns := []struct {
		account int64
		kind    string
		amount  int64
		day     int
	}{
		{checkingID, "income", 250, -25},
		{checkingID, "expense", 80, -20},
		{checkingID, "expense", 40, -10},
		{cardID, "expense", 150, -18},
		{cardID, "income", 200, -8},
	}
	for _, t := range txns {
		_, err := s.TC.App.TransactionService.InsertTransaction(s.Ctx, seedUserID, &models.TransactionReq{
			AccountID: t.account,
			Direction: t.kind,
			Amount:    decimal.NewFromInt(t.amount),
			TxnDate:   today.AddDate(0, 0, t.day),
		})
		s.Require().NoError(err)
	}

	assetID, err := s.TC.App.InvestmentService.InsertAsset(s.Ctx, seedUserID, &models.InvestmentAssetReq{
		AccountID:      brokerID,
		InvestmentType: models.InvestmentETF,
		Name:           "iShares Core MSCI World",
		Ticker:         "IWDA.AS",
		Quantity:       decimal.Zero,
		Currency:       "EUR",
	})
	s.Require().NoError(err)

	fee := decimal.NewFromInt(2)
	trades := []struct {
		kind models.TradeType
		qty  int64
		unit int64
		day  int
	}{
		{models.InvestmentBuy, 10, 100, -22},
		{models.InvestmentSell, 4, 110, -12},
	}
	for _, t := range trades {
		_, err := s.TC.App.InvestmentService.InsertInvestmentTrade(s.Ctx, seedUserID, &models.InvestmentTradeReq{
			AssetID:      assetID,
			TradeType:    t.kind,
			TxnDate:      today.AddDate(0, 0, t.day),
			Quantity:     decimal.NewFromInt(t.qty),
			PricePerUnit: decimal.NewFromInt(t.unit),
			Currency:     "EUR",
			Fee:          &fee,
		})
		s.Require().NoError(err)
	}

	return []int64{checkingID, brokerID, cardID}
}

func (s *BalanceServiceSuite) TestApplyDeltaAccumulates() {
	ids := s.seedAccounts()
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
func (s *BalanceServiceSuite) TestApplyDeltaSeedsAMissingRow() {
	ids := s.seedAccounts()
	repo := repositories.NewBalanceRepository(s.TC.DB)

	id := ids[0]
	s.Require().NoError(s.TC.DB.Exec(`DELETE FROM balances WHERE account_id = ?`, id).Error)

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
func (s *BalanceServiceSuite) assertBalanceMatchesLedger(context string) {
	type row struct {
		AccountID int64
		Balance   decimal.Decimal
		TxnSum    decimal.Decimal
		Openings  int64
	}
	var rows []row
	s.Require().NoError(s.TC.DB.Raw(`
		SELECT a.id AS account_id,
		       COALESCE(ab.balance, 0) AS balance,
		       COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0) AS txn_sum,
		       COUNT(*) FILTER (WHERE t.transaction_type = 'opening'
		                          AND t.category_id IS NOT NULL) AS openings
		FROM accounts a
		LEFT JOIN balances ab ON ab.account_id = a.id
		LEFT JOIN transactions t ON t.account_id = a.id AND t.deleted_at IS NULL
		GROUP BY a.id, ab.balance
		ORDER BY a.id`).Scan(&rows).Error)

	s.Require().NotEmpty(rows)
	for _, r := range rows {
		s.Assert().True(r.Balance.Round(4).Equal(r.TxnSum.Round(4)),
			"%s: account %d holds %s, its transactions sum to %s",
			context, r.AccountID, r.Balance.StringFixed(4), r.TxnSum.StringFixed(4))
		s.Assert().EqualValues(1, r.Openings,
			"%s: account %d needs exactly one opening transaction, with a category", context, r.AccountID)
	}
}

func (s *BalanceServiceSuite) TestWritePathsKeepTheOneRowBalance() {
	today := s.today()
	ids := s.seedAccounts()
	checkingID, brokerID, cardID := ids[0], ids[1], ids[2]

	// The seeders go through the same seams, so the seed alone must already agree.
	s.assertBalanceMatchesLedger("after the seed")

	s.Run("transaction insert", func() {
		amount := decimal.NewFromInt(64)
		_, err := s.TC.App.TransactionService.InsertTransaction(s.Ctx, seedUserID, &models.TransactionReq{
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

		_, err := s.TC.App.TransactionService.UpdateTransaction(s.Ctx, seedUserID, txn.ID, &models.TransactionReq{
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

		s.Require().NoError(s.TC.App.TransactionService.DeleteTransaction(s.Ctx, seedUserID, id))
		s.assertBalanceMatchesLedger("after a transaction delete")
	})

	s.Run("transfer", func() {
		_, err := s.TC.App.TransactionService.InsertTransfer(s.Ctx, seedUserID, &models.TransferReq{
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
		_, err := s.TC.App.AccountService.UpdateAccount(s.Ctx, seedUserID, checkingID, &models.AccountReq{
			Name:          "Checking",
			AccountTypeID: typeID,
			Balance:       &desired,
		})
		s.Require().NoError(err)
		s.assertBalanceMatchesLedger("after a manual balance edit")
	})

	s.Run("dividend", func() {
		var assetID int64
		s.Require().NoError(s.TC.DB.Raw(
			`SELECT id FROM investment_assets WHERE account_id = ?`, brokerID).Scan(&assetID).Error)
		s.Require().NotZero(assetID)

		amount := decimal.NewFromInt(30)
		tax := decimal.NewFromInt(5)
		_, err := s.TC.App.InvestmentService.CreateInvestmentIncome(s.Ctx, seedUserID, &models.InvestmentIncomeReq{
			AssetID:     assetID,
			TxnDate:     today.AddDate(0, 0, -6),
			IncomeType:  models.IncomeTypeDividend,
			Amount:      &amount,
			TaxWithheld: &tax,
			Currency:    "EUR",
		})
		s.Require().NoError(err)
		s.assertBalanceMatchesLedger("after dividend income")
	})

	s.Run("trade delete", func() {
		var id int64
		s.Require().NoError(s.TC.DB.Raw(`
			SELECT it.id FROM investment_trades it
			JOIN investment_assets ia ON ia.id = it.asset_id
			WHERE ia.account_id = ? AND it.trade_type = 'sell'
			ORDER BY it.id LIMIT 1`, brokerID).Scan(&id).Error)
		s.Require().NotZero(id)

		s.Require().NoError(s.TC.App.InvestmentService.DeleteInvestmentTrade(s.Ctx, seedUserID, id))
		s.assertBalanceMatchesLedger("after a trade delete")
	})
}

func (s *BalanceServiceSuite) service() *services.BalanceService {
	return services.NewBalanceService(zap.NewNop(), repositories.NewBalanceRepository(s.TC.DB))
}

// Walks the pages the way the nightly parent job does, then reconciles each page the
// way a child job does. Closed accounts are filtered by the listing, not the repair.
func (s *BalanceServiceSuite) reconcileAll() ([]models.BalanceDrift, error) {
	svc := s.service()
	var repaired []models.BalanceDrift

	for afterID := int64(0); ; {
		ids, err := svc.ListOpenAccountIDs(s.Ctx, afterID, 500)
		if err != nil {
			return repaired, err
		}
		if len(ids) == 0 {
			return repaired, nil
		}
		afterID = ids[len(ids)-1]

		batch, err := svc.ReconcileAccounts(s.Ctx, ids)
		repaired = append(repaired, batch...)
		if err != nil {
			return repaired, err
		}
	}
}

func (s *BalanceServiceSuite) balanceOf(accountID int64) decimal.Decimal {
	var balance decimal.Decimal
	s.Require().NoError(s.TC.DB.
		Raw(`SELECT balance FROM balances WHERE account_id = ?`, accountID).
		Scan(&balance).Error)
	return balance
}

func (s *BalanceServiceSuite) TestReconcileRepairsOnlyTheDriftedAccount() {
	ids := s.seedAccounts()

	broken, untouched := ids[0], ids[1]
	want := s.balanceOf(broken)
	wantUntouched := s.balanceOf(untouched)

	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE balances SET balance = balance + 42.5 WHERE account_id = ?`, broken).Error)

	repaired, err := s.reconcileAll()
	s.Require().NoError(err)

	s.Require().Len(repaired, 1)
	s.Assert().Equal(broken, repaired[0].AccountID)
	s.Assert().NotZero(repaired[0].UserID)
	s.Assert().True(repaired[0].Actual.Equal(want.Add(decimal.NewFromFloat(42.5))),
		"actual was %s", repaired[0].Actual.StringFixed(4))
	s.Assert().True(repaired[0].Expected.Equal(want), "expected was %s", repaired[0].Expected.StringFixed(4))
	s.Assert().True(repaired[0].Difference().Equal(decimal.NewFromFloat(-42.5)),
		"difference was %s", repaired[0].Difference().StringFixed(4))

	s.Assert().True(s.balanceOf(broken).Equal(want), "the drifted balance was not repaired")
	s.Assert().True(s.balanceOf(untouched).Equal(wantUntouched), "a sound balance was rewritten")
}

// A missing row is drift too, and it has nothing to lock until the repair makes one.
func (s *BalanceServiceSuite) TestReconcileRebuildsAMissingBalanceRow() {
	ids := s.seedAccounts()

	orphan := ids[0]
	want := s.balanceOf(orphan)
	s.Require().False(want.IsZero(), "the seeded account needs a non-zero balance")

	s.Require().NoError(s.TC.DB.
		Exec(`DELETE FROM balances WHERE account_id = ?`, orphan).Error)

	repaired, err := s.reconcileAll()
	s.Require().NoError(err)

	s.Require().Len(repaired, 1)
	s.Assert().Equal(orphan, repaired[0].AccountID)
	s.Assert().True(s.balanceOf(orphan).Equal(want), "the missing row was not rebuilt")
}

// A closed account is out of scope: nothing can post to it and no read shows it.
func (s *BalanceServiceSuite) TestReconcileSkipsClosedAccounts() {
	ids := s.seedAccounts()

	closed := ids[0]
	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE accounts SET closed_at = now(), is_active = false WHERE id = ?`, closed).Error)
	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE balances SET balance = balance + 99 WHERE account_id = ?`, closed).Error)

	repaired, err := s.reconcileAll()
	s.Require().NoError(err)
	s.Assert().Empty(repaired)
}

func (s *BalanceServiceSuite) TestReconcileReportsNothingWhenBalancesAgree() {
	s.seedAccounts()

	repaired, err := s.reconcileAll()
	s.Require().NoError(err)
	s.Assert().Empty(repaired)
}
