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
