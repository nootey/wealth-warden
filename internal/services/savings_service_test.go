package services_test

import (
	"fmt"
	"testing"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm/clause"
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
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
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
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
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
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
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
	_, err = s.TC.App.TransactionService.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID: accID,
		Direction: "income",
		Amount:    decimal.NewFromInt(500),
		TxnDate:   time.Now().UTC(),
	})
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

// FundGoalNow funds a goal on demand and blocks a second call the same month, but a manual
// contribution afterward is unaffected by (and does not satisfy) that guard.
func (s *SavingsServiceTestSuite) TestFundGoalNow_BlocksSecondCallSameMonth_ButAllowsManualContribution() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
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

	err = svc.FundGoalNow(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	// Second on-demand call the same month is blocked by AutoFundGoal's guard.
	err = svc.FundGoalNow(s.Ctx, userID, goalID)
	s.Require().Error(err)
	s.Contains(err.Error(), "already funded")

	// A manual contribution is still allowed and does not get blocked by the auto guard.
	_, err = svc.InsertContribution(s.Ctx, userID, goalID, &models.SavingContributionReq{
		Amount: decimal.NewFromInt(50),
		Month:  time.Now().UTC().Format("2006-01-02"),
	})
	s.Require().NoError(err)

	var count int64
	s.Require().NoError(
		s.TC.DB.Model(&models.SavingContribution{}).
			Where("goal_id = ?", goalID).
			Count(&count).Error,
	)
	s.Equal(int64(2), count)
}

// InsertContribution rejects contributions (positive and negative) for non-active goals.
func (s *SavingsServiceTestSuite) TestInsertContribution_NonActiveGoalRejected() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
	})
	s.Require().NoError(err)

	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:    accID,
		Name:         "Test Goal",
		TargetAmount: decimal.NewFromInt(1000),
	})
	s.Require().NoError(err)

	nonActiveStatuses := []models.SavingGoalStatus{
		models.SavingGoalStatusPaused,
		models.SavingGoalStatusCompleted,
		models.SavingGoalStatusArchived,
	}

	for _, status := range nonActiveStatuses {
		_, err = svc.UpdateGoal(s.Ctx, userID, goalID, &models.SavingGoalUpdateReq{
			Name:         "Test Goal",
			TargetAmount: decimal.NewFromInt(1000),
			Status:       status,
		})
		s.Require().NoError(err)

		_, err = svc.InsertContribution(s.Ctx, userID, goalID, &models.SavingContributionReq{
			Amount: decimal.NewFromInt(50),
			Month:  "2026-04-01",
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "cannot add contributions")

		_, err = svc.InsertContribution(s.Ctx, userID, goalID, &models.SavingContributionReq{
			Amount: decimal.NewFromInt(-50),
			Month:  "2026-04-01",
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "cannot add contributions")
	}

	var count int64
	s.Require().NoError(
		s.TC.DB.Model(&models.SavingContribution{}).
			Where("goal_id = ?", goalID).
			Count(&count).Error,
	)
	s.Equal(int64(0), count)

	// Reactivated goal accepts contributions again
	_, err = svc.UpdateGoal(s.Ctx, userID, goalID, &models.SavingGoalUpdateReq{
		Name:         "Test Goal",
		TargetAmount: decimal.NewFromInt(1000),
		Status:       models.SavingGoalStatusActive,
	})
	s.Require().NoError(err)

	_, err = svc.InsertContribution(s.Ctx, userID, goalID, &models.SavingContributionReq{
		Amount: decimal.NewFromInt(50),
		Month:  "2026-04-01",
	})
	s.Require().NoError(err)
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
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
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

// An initial allocation above the linked account's balance is rejected.
func (s *SavingsServiceTestSuite) TestInsertGoal_InitialAmountExceedsBalance() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
	})
	s.Require().NoError(err)

	initial := decimal.NewFromInt(600)
	_, err = svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:     accID,
		Name:          "Overfunded Goal",
		TargetAmount:  decimal.NewFromInt(1000),
		InitialAmount: &initial,
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "exceeds uncategorized balance")

	goals, err := svc.FetchGoals(s.Ctx, userID)
	s.Require().NoError(err)
	s.Empty(goals)
}

// An initial allocation is measured against the balance left over by goals already on the account.
func (s *SavingsServiceTestSuite) TestInsertGoal_InitialAmountRespectsSiblingAllocations() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Test Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
	})
	s.Require().NoError(err)

	firstInitial := decimal.NewFromInt(400)
	_, err = svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:     accID,
		Name:          "First Goal",
		TargetAmount:  decimal.NewFromInt(1000),
		InitialAmount: &firstInitial,
	})
	s.Require().NoError(err)

	// Only 100 remains uncategorized on the account.
	secondInitial := decimal.NewFromInt(150)
	_, err = svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:     accID,
		Name:          "Second Goal",
		TargetAmount:  decimal.NewFromInt(1000),
		InitialAmount: &secondInitial,
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "exceeds uncategorized balance")

	withinRemaining := decimal.NewFromInt(100)
	_, err = svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:     accID,
		Name:          "Second Goal",
		TargetAmount:  decimal.NewFromInt(1000),
		InitialAmount: &withinRemaining,
	})
	s.Require().NoError(err)
}

func (s *SavingsServiceTestSuite) TestAutoFundGoal_ConcurrentRunsFundOnce() {
	svc := s.TC.App.SavingsService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	balance := decimal.NewFromInt(500)
	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:           "Race Savings",
		AccountTypeID:  2,
		Type:           "cash",
		Subtype:        "savings",
		Classification: "asset",
		Balance:        &balance,
		OpenedAt:       time.Now(),
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(100)
	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Race Goal",
		TargetAmount:      decimal.NewFromInt(1000),
		MonthlyAllocation: &alloc,
	})
	s.Require().NoError(err)

	goalWithProgress, err := svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)
	goal := goalWithProgress.SavingGoal

	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	blocker := s.TC.DB.WithContext(s.Ctx).Begin()
	s.Require().NoError(blocker.Error)
	var locked models.SavingGoal
	s.Require().NoError(blocker.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", goalID).First(&locked).Error)

	var funded bool
	var reason string
	done := make(chan error, 1)
	go func() {
		var runErr error
		funded, reason, runErr = svc.AutoFundGoal(s.Ctx, goal, month)
		done <- runErr
	}()

	time.Sleep(300 * time.Millisecond)

	s.Require().NoError(blocker.Create(&models.SavingContribution{
		UserID: userID,
		GoalID: goalID,
		Amount: alloc,
		Month:  month,
		Source: models.SavingContributionSourceAuto,
	}).Error)
	s.Require().NoError(blocker.Commit().Error)

	s.Require().NoError(<-done)
	s.False(funded, "the second runner must not fund a goal already funded this month")
	s.Equal("already_funded", reason)

	var contributions int64
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Model(&models.SavingContribution{}).
		Where("goal_id = ? AND month = ?", goalID, month).Count(&contributions).Error)
	s.Equal(int64(1), contributions, "the month must hold a single contribution")
}

func (s *SavingsServiceTestSuite) newSavingsAccount(name string, balance decimal.Decimal) int64 {
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, int64(1), &models.AccountReq{
		Name:          name,
		AccountTypeID: 2,
		Balance:       &balance,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)
	return accID
}

func (s *SavingsServiceTestSuite) TestInsertContribution_ConcurrentContributionsAllCount() {
	svc := s.TC.App.SavingsService
	userID := int64(1)
	accID := s.newSavingsAccount("Contribution Race", decimal.NewFromInt(10000))

	goalID, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:    accID,
		Name:         "Race Goal",
		TargetAmount: decimal.NewFromInt(5000),
	})
	s.Require().NoError(err)

	errs := s.RunConcurrently(20, func(int) error {
		_, err := svc.InsertContribution(s.Ctx, userID, goalID, &models.SavingContributionReq{
			Amount: decimal.NewFromInt(10),
			Month:  "2026-05-01",
		})
		return err
	})
	for i, err := range errs {
		s.Require().NoError(err, "call %d", i)
	}

	goal, err := svc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(200).Equal(goal.CurrentAmount), "allocation is %s, want 200", goal.CurrentAmount)
}

func (s *SavingsServiceTestSuite) TestInsertContribution_RacingExpensesNeverOverAllocate() {
	svc := s.TC.App.SavingsService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)
	accID := s.newSavingsAccount("Allocation Race", decimal.NewFromInt(1000))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// One goal per contribution, so the race is over the account and not a goal row
	goalIDs := make([]int64, 5)
	for i := range goalIDs {
		id, err := svc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
			AccountID:    accID,
			Name:         fmt.Sprintf("Race Goal %d", i),
			TargetAmount: decimal.NewFromInt(1000),
		})
		s.Require().NoError(err)
		goalIDs[i] = id
	}

	errs := s.RunConcurrently(10, func(i int) error {
		if i%2 == 0 {
			_, err := txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
				AccountID: accID, Direction: "expense", Amount: decimal.NewFromInt(200), TxnDate: today,
			})
			return err
		}
		_, err := svc.InsertContribution(s.Ctx, userID, goalIDs[i/2], &models.SavingContributionReq{
			Amount: decimal.NewFromInt(200),
			Month:  "2026-05-01",
		})
		return err
	})

	accepted := 0
	for _, err := range errs {
		if err == nil {
			accepted++
			continue
		}
		var appErr *apperr.Error
		s.Require().ErrorAs(err, &appErr, "only free-balance rejections may fail")
		s.Require().Equal(apperr.Validation, appErr.Kind, "unexpected error: %v", err)
	}
	// Each accepted call takes 200 of the 1000 free, whatever the order
	s.Assert().Equal(5, accepted, "accepted calls")

	var balance, allocated decimal.Decimal
	s.Require().NoError(s.TC.DB.Raw(`SELECT balance FROM balances WHERE account_id = ?`, accID).Scan(&balance).Error)
	s.Require().NoError(s.TC.DB.Raw(`SELECT COALESCE(SUM(current_amount), 0) FROM saving_goals WHERE account_id = ?`, accID).Scan(&allocated).Error)
	s.Assert().True(allocated.LessThanOrEqual(balance), "allocated %s exceeds balance %s", allocated, balance)
}
