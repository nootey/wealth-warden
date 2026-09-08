package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type BalanceReconcileIntegrationSuite struct {
	tests.ServiceIntegrationSuite
}

func TestBalanceReconcileIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BalanceReconcileIntegrationSuite))
}

func (s *BalanceReconcileIntegrationSuite) seedAccounts() []int64 {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := tests.SeedBalanceFixture(s.Ctx, s.TC.App, s.TC.DB, today)
	s.Require().NoError(err)

	var ids []int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM accounts ORDER BY id`).Scan(&ids).Error)
	s.Require().True(len(ids) > 1)
	return ids
}

func (s *BalanceReconcileIntegrationSuite) service() *services.BalanceService {
	return services.NewBalanceService(zap.NewNop(), repositories.NewBalanceRepository(s.TC.DB))
}

func (s *BalanceReconcileIntegrationSuite) balanceOf(accountID int64) decimal.Decimal {
	var balance decimal.Decimal
	s.Require().NoError(s.TC.DB.
		Raw(`SELECT balance FROM balances WHERE account_id = ?`, accountID).
		Scan(&balance).Error)
	return balance
}

func (s *BalanceReconcileIntegrationSuite) TestReconcileRepairsOnlyTheDriftedAccount() {
	ids := s.seedAccounts()

	broken, untouched := ids[0], ids[1]
	want := s.balanceOf(broken)
	wantUntouched := s.balanceOf(untouched)

	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE balances SET balance = balance + 42.5 WHERE account_id = ?`, broken).Error)

	repaired, err := s.service().ReconcileBalances(s.Ctx)
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
func (s *BalanceReconcileIntegrationSuite) TestReconcileRebuildsAMissingBalanceRow() {
	ids := s.seedAccounts()

	orphan := ids[0]
	want := s.balanceOf(orphan)
	s.Require().False(want.IsZero(), "the fixture account needs a non-zero balance")

	s.Require().NoError(s.TC.DB.
		Exec(`DELETE FROM balances WHERE account_id = ?`, orphan).Error)

	repaired, err := s.service().ReconcileBalances(s.Ctx)
	s.Require().NoError(err)

	s.Require().Len(repaired, 1)
	s.Assert().Equal(orphan, repaired[0].AccountID)
	s.Assert().True(s.balanceOf(orphan).Equal(want), "the missing row was not rebuilt")
}

// A closed account is out of scope: nothing can post to it and no read shows it.
func (s *BalanceReconcileIntegrationSuite) TestReconcileSkipsClosedAccounts() {
	ids := s.seedAccounts()

	closed := ids[0]
	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE accounts SET closed_at = now(), is_active = false WHERE id = ?`, closed).Error)
	s.Require().NoError(s.TC.DB.
		Exec(`UPDATE balances SET balance = balance + 99 WHERE account_id = ?`, closed).Error)

	repaired, err := s.service().ReconcileBalances(s.Ctx)
	s.Require().NoError(err)
	s.Assert().Empty(repaired)
}

func (s *BalanceReconcileIntegrationSuite) TestReconcileReportsNothingWhenBalancesAgree() {
	s.seedAccounts()

	repaired, err := s.service().ReconcileBalances(s.Ctx)
	s.Require().NoError(err)
	s.Assert().Empty(repaired)
}
