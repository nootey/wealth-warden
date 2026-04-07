package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type SavingsServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestSavingsServiceSuite(t *testing.T) {
	suite.Run(t, new(SavingsServiceTestSuite))
}

// AutoFundGoal called twice for the same month returns already_funded on the second call.
func (s *SavingsServiceTestSuite) TestAutoFundGoal_SkipReason_AlreadyFunded() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Test Savings",
		AccountTypeID: 2,
		Type:          "cash",
		Subtype:       "savings",
		Classification: "asset",
		Balance:       &balance,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(100)
	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Test Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
	})
	s.Require().NoError(err)

	goalWithProgress, err := svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)
	goal := goalWithProgress.SavingGoal

	month := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	funded, reason, err := svc.AutoFundGoal(s.Ctx, goal, month)
	s.Require().NoError(err)
	s.True(funded)
	s.Empty(reason)

	// Re-fetch goal to get updated current_amount
	goalWithProgress, err = svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	funded, reason, err = svc.AutoFundGoal(s.Ctx, goalWithProgress.SavingGoal, month)
	s.Require().NoError(err)
	s.False(funded)
	s.Equal("already_funded", reason)
}

// AutoFundGoal returns insufficient_balance when uncategorized balance is below the allocation.
func (s *SavingsServiceTestSuite) TestAutoFundGoal_SkipReason_InsufficientBalance() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(50)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Test Savings",
		AccountTypeID: 2,
		Type:          "cash",
		Subtype:       "savings",
		Classification: "asset",
		Balance:       &balance,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(100)
	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Test Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
	})
	s.Require().NoError(err)

	goalWithProgress, err := svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	funded, reason, err := svc.AutoFundGoal(s.Ctx, goalWithProgress.SavingGoal, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.False(funded)
	s.Equal("insufficient_balance", reason)
}

// A month with no balance is never retroactively funded when balance is credited the following month.
func (s *SavingsServiceTestSuite) TestAutoFundGoal_MissedMonthStaysMissed() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(0)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Test Savings",
		AccountTypeID: 2,
		Type:          "cash",
		Subtype:       "savings",
		Classification: "asset",
		Balance:       &balance,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(100)
	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Test Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
	})
	s.Require().NoError(err)

	goalWithProgress, err := svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	monthA := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	monthB := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Month A: no balance - skipped
	funded, _, err := svc.AutoFundGoal(s.Ctx, goalWithProgress.SavingGoal, monthA)
	s.Require().NoError(err)
	s.False(funded)

	// Credit balance in time for month B
	err = s.TC.DB.Exec("UPDATE balances SET start_balance = 500 WHERE account_id = ?", accID).Error
	s.Require().NoError(err)

	goalWithProgress, err = svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	// Month B: funded
	funded, _, err = svc.AutoFundGoal(s.Ctx, goalWithProgress.SavingGoal, monthB)
	s.Require().NoError(err)
	s.True(funded)

	// Month A must still have no contribution
	var countA int64
	s.Require().NoError(
		s.TC.DB.Model(&models.SavingContribution{}).
			Where("goal_id = ? AND month = ?", goalID, monthA).
			Count(&countA).Error,
	)
	s.Equal(int64(0), countA)

	// Month B must have exactly one contribution
	var countB int64
	s.Require().NoError(
		s.TC.DB.Model(&models.SavingContribution{}).
			Where("goal_id = ? AND month = ?", goalID, monthB).
			Count(&countB).Error,
	)
	s.Equal(int64(1), countB)
}

// FetchActiveGoalsWithAllocation excludes goals whose fund_day_of_month is in the future.
func (s *SavingsServiceTestSuite) TestFetchActiveGoalsWithAllocation_DayFilter() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Day()
	if today == 31 {
		s.T().Skip("no future day available on day 31")
	}

	balance := decimal.NewFromInt(1000)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Test Savings",
		AccountTypeID: 2,
		Type:          "cash",
		Subtype:       "savings",
		Classification: "asset",
		Balance:       &balance,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(100)
	pastDay := 1
	futureDay := today + 1

	pastGoalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Past Day Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
		FundDayOfMonth:    &pastDay,
	})
	s.Require().NoError(err)

	_, err = svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Future Day Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
		FundDayOfMonth:    &futureDay,
	})
	s.Require().NoError(err)

	goals, err := svc.FetchActiveGoalsWithAllocation(s.Ctx, today)
	s.Require().NoError(err)

	ids := make([]int64, len(goals))
	for i, g := range goals {
		ids[i] = g.ID
	}

	s.Contains(ids, pastGoalID)
	for _, g := range goals {
		if g.FundDayOfMonth != nil {
			s.LessOrEqual(*g.FundDayOfMonth, today)
		}
	}
}
