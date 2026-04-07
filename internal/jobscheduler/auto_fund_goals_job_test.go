package jobscheduler_test

import (
	"testing"
	"time"
	"wealth-warden/internal/jobscheduler"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type AutoFundGoalsJobTestSuite struct {
	tests.ServiceIntegrationSuite
	memberUserID  int64
	savingsTypeID int64
}

func TestAutoFundGoalsJobSuite(t *testing.T) {
	suite.Run(t, new(AutoFundGoalsJobTestSuite))
}

func (s *AutoFundGoalsJobTestSuite) SetupSuite() {
	s.ServiceIntegrationSuite.SetupSuite()

	var user models.User
	err := s.TC.DB.Where("role_id = ?", 1).First(&user).Error
	s.Require().NoError(err)
	s.memberUserID = user.ID

	var at models.AccountType
	err = s.TC.DB.Where("sub_type = ?", "savings").First(&at).Error
	s.Require().NoError(err)
	s.savingsTypeID = at.ID
}

// createSavingsAccount creates an account with a balance row whose end_balance equals the given amount.
// end_balance is generated as start_balance + inflows - outflows, so setting start_balance is sufficient.
func (s *AutoFundGoalsJobTestSuite) createSavingsAccount(name string, balance decimal.Decimal) models.Account {
	acc := models.Account{
		UserID:            s.memberUserID,
		Name:              name,
		AccountTypeID:     s.savingsTypeID,
		Currency:          "EUR",
		BalanceProjection: "fixed",
		ExpectedBalance:   decimal.Zero,
		OpenedAt:          time.Now().UTC(),
		IsActive:          true,
	}
	s.Require().NoError(s.TC.DB.Create(&acc).Error)

	bal := models.Balance{
		AccountID:    acc.ID,
		AsOf:         time.Now().UTC(),
		StartBalance: balance,
		Currency:     "EUR",
	}
	s.Require().NoError(s.TC.DB.Create(&bal).Error)

	return acc
}

func (s *AutoFundGoalsJobTestSuite) createGoal(accountID int64, allocation decimal.Decimal, priority int, fundDayOfMonth *int) models.SavingGoal {
	goal := models.SavingGoal{
		UserID:            s.memberUserID,
		AccountID:         accountID,
		Name:              "Test Goal",
		TargetAmount:      decimal.NewFromInt(10000),
		Status:            models.SavingGoalStatusActive,
		Priority:          priority,
		MonthlyAllocation: &allocation,
		FundDayOfMonth:    fundDayOfMonth,
	}
	s.Require().NoError(s.TC.DB.Create(&goal).Error)
	return goal
}

func (s *AutoFundGoalsJobTestSuite) countContributions(goalID int64) int64 {
	var count int64
	s.Require().NoError(
		s.TC.DB.Model(&models.SavingContribution{}).Where("goal_id = ?", goalID).Count(&count).Error,
	)
	return count
}

func (s *AutoFundGoalsJobTestSuite) runJob() {
	logger := zaptest.NewLogger(s.T())
	job := jobscheduler.NewAutoFundGoalsJob(logger, s.TC.App, 2)
	s.Require().NoError(job.Run(s.Ctx))
}

// Both goals have sufficient balance and are funded in priority order.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_HappyPath() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))

	alloc100 := decimal.NewFromInt(100)
	alloc200 := decimal.NewFromInt(200)
	goal1 := s.createGoal(acc.ID, alloc100, 10, nil)
	goal2 := s.createGoal(acc.ID, alloc200, 5, nil)

	s.runJob()

	s.Equal(int64(1), s.countContributions(goal1.ID))
	s.Equal(int64(1), s.countContributions(goal2.ID))
}

// Higher priority goal cannot be funded - all lower priority goals on the same account are also blocked.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_InsufficientBalance_StopsChain() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(400))

	alloc500 := decimal.NewFromInt(500)
	alloc100 := decimal.NewFromInt(100)
	goal1 := s.createGoal(acc.ID, alloc500, 10, nil) // needs 500, only 400 uncategorized
	goal2 := s.createGoal(acc.ID, alloc100, 5, nil)  // would fit, but blocked by goal1

	s.runJob()

	s.Equal(int64(0), s.countContributions(goal1.ID))
	s.Equal(int64(0), s.countContributions(goal2.ID))
}

// Account balance equals what is already allocated across goals - uncategorized balance is 0.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_BalanceBelowAllocated() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(500))

	alloc100 := decimal.NewFromInt(100)
	goal := s.createGoal(acc.ID, alloc100, 0, nil)

	// Simulate all balance already being allocated to this goal
	err := s.TC.DB.Model(&models.SavingGoal{}).
		Where("id = ?", goal.ID).
		Update("current_amount", decimal.NewFromInt(500)).Error
	s.Require().NoError(err)

	s.runJob()

	s.Equal(int64(0), s.countContributions(goal.ID))
}

// Goal with fund_day_of_month set to a future day is excluded from the job run entirely.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_FutureDaySkipped() {
	today := time.Now().UTC().Day()
	if today == 31 {
		s.T().Skip("no future day available on day 31")
	}
	futureDay := today + 1

	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))
	goal := s.createGoal(acc.ID, decimal.NewFromInt(100), 0, &futureDay)

	s.runJob()

	s.Equal(int64(0), s.countContributions(goal.ID))
}

// Goal with fund_day_of_month set to a past day this month is still funded (catch-up behaviour).
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_PastDayCatchup() {
	today := time.Now().UTC().Day()
	if today == 1 {
		s.T().Skip("no past day available on day 1")
	}
	pastDay := 1

	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))
	goal := s.createGoal(acc.ID, decimal.NewFromInt(100), 0, &pastDay)

	s.runJob()

	s.Equal(int64(1), s.countContributions(goal.ID))
}

// Running the job twice in the same month must not create duplicate contributions.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_Idempotency() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))
	goal := s.createGoal(acc.ID, decimal.NewFromInt(100), 0, nil)

	s.runJob()
	s.runJob()

	s.Equal(int64(1), s.countContributions(goal.ID))
}

// When balance only covers one goal, the higher priority goal is funded and the lower one is blocked.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_PriorityOrdering() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(300))

	alloc200 := decimal.NewFromInt(200)
	alloc200b := decimal.NewFromInt(200)
	highPriority := s.createGoal(acc.ID, alloc200, 10, nil) // funded - 300 >= 200
	lowPriority := s.createGoal(acc.ID, alloc200b, 5, nil)  // blocked - only 100 left after high priority

	s.runJob()

	s.Equal(int64(1), s.countContributions(highPriority.ID))
	s.Equal(int64(0), s.countContributions(lowPriority.ID))
}

// A goal with status != active is not processed even if it has a monthly allocation.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_InactiveGoalSkipped() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))
	goal := s.createGoal(acc.ID, decimal.NewFromInt(100), 0, nil)

	err := s.TC.DB.Model(&models.SavingGoal{}).
		Where("id = ?", goal.ID).
		Update("status", models.SavingGoalStatusPaused).Error
	s.Require().NoError(err)

	s.runJob()

	s.Equal(int64(0), s.countContributions(goal.ID))
}

// A funding failure on one account must not affect goals on a separate account.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_SeparateAccountsAreIndependent() {
	accA := s.createSavingsAccount(s.T().Name()+"-A", decimal.NewFromInt(50))
	accB := s.createSavingsAccount(s.T().Name()+"-B", decimal.NewFromInt(500))

	goalA := s.createGoal(accA.ID, decimal.NewFromInt(200), 0, nil) // will be skipped
	goalB := s.createGoal(accB.ID, decimal.NewFromInt(200), 0, nil) // should still be funded

	s.runJob()

	s.Equal(int64(0), s.countContributions(goalA.ID))
	s.Equal(int64(1), s.countContributions(goalB.ID))
}

// Manual contributions for the same month must not block the auto-fund job.
func (s *AutoFundGoalsJobTestSuite) TestFundGoals_ManualContributionDoesNotBlockAutoFund() {
	acc := s.createSavingsAccount(s.T().Name(), decimal.NewFromInt(1000))
	goal := s.createGoal(acc.ID, decimal.NewFromInt(100), 0, nil)

	// Insert a manual contribution for the current month
	monthStart := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	manual := models.SavingContribution{
		UserID: s.memberUserID,
		GoalID: goal.ID,
		Amount: decimal.NewFromInt(50),
		Month:  monthStart,
		Source: models.SavingContributionSourceManual,
	}
	s.Require().NoError(s.TC.DB.Create(&manual).Error)

	s.runJob()

	// Both manual and auto contributions should exist
	s.Equal(int64(2), s.countContributions(goal.ID))
}
