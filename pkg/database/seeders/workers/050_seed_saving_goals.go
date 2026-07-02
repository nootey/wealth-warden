package workers

import (
	"context"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func SeedSavingGoals(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	type goalSeed struct {
		Name              string
		Status            models.SavingGoalStatus
		TargetAmount      decimal.Decimal
		TargetDate        time.Time
		CreatedMonthsAgo  int
		InitialAmount     decimal.Decimal
		MonthlyAllocation decimal.Decimal // zero = no allocation
		Priority          int
	}

	// Amounts are tuned against computeProgress: with these creation dates and
	// targets, "Emergency fund" lands ahead of schedule and "Vacation" on track
	seeds := []goalSeed{
		{Name: "Emergency fund", Status: models.SavingGoalStatusActive, TargetAmount: decimal.NewFromInt(4000), TargetDate: now.AddDate(0, 6, 0), CreatedMonthsAgo: 6, InitialAmount: decimal.NewFromInt(600), MonthlyAllocation: decimal.NewFromInt(400), Priority: 0},
		{Name: "Vacation", Status: models.SavingGoalStatusActive, TargetAmount: decimal.NewFromInt(3000), TargetDate: now.AddDate(0, 6, 0), CreatedMonthsAgo: 6, InitialAmount: decimal.NewFromInt(750), MonthlyAllocation: decimal.NewFromInt(120), Priority: 1},
		{Name: "New car", Status: models.SavingGoalStatusPaused, TargetAmount: decimal.NewFromInt(15000), TargetDate: now.AddDate(0, 18, 0), CreatedMonthsAgo: 8, InitialAmount: decimal.NewFromInt(2500), Priority: 2},
	}

	// Total that would end up allocated with an unscaled seed set
	plannedTotal := decimal.Zero
	for _, s := range seeds {
		total := s.InitialAmount
		if s.MonthlyAllocation.IsPositive() {
			total = total.Add(s.MonthlyAllocation.Mul(decimal.NewFromInt(int64(s.CreatedMonthsAgo))))
		}
		plannedTotal = plannedTotal.Add(total)
	}

	var users []models.User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		return err
	}

	for _, u := range users {
		var acc models.Account
		err := db.WithContext(ctx).
			Where("user_id = ? AND name = ?", u.ID, "Savings account").
			First(&acc).Error
		if err == gorm.ErrRecordNotFound {
			continue
		}
		if err != nil {
			return err
		}

		var bal models.Balance
		if err := db.WithContext(ctx).
			Where("account_id = ?", acc.ID).
			Order("as_of DESC").
			First(&bal).Error; err != nil {
			return err
		}

		// Keep total allocations within the account so the uncategorized
		// balance stays positive; scale everything down proportionally
		budget := bal.EndBalance.Mul(decimal.NewFromFloat(0.7))
		if budget.LessThan(decimal.NewFromInt(100)) {
			continue
		}
		scale := decimal.NewFromInt(1)
		if plannedTotal.GreaterThan(budget) {
			scale = budget.Div(plannedTotal)
		}

		for _, s := range seeds {
			var existing models.SavingGoal
			err := db.WithContext(ctx).
				Where("user_id = ? AND name = ?", u.ID, s.Name).
				First(&existing).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			createdAt := currentMonth.AddDate(0, -s.CreatedMonthsAgo, 0)
			targetDate := s.TargetDate

			goal := models.SavingGoal{
				UserID:       u.ID,
				AccountID:    acc.ID,
				Name:         s.Name,
				TargetAmount: s.TargetAmount.Mul(scale).Round(2),
				TargetDate:   &targetDate,
				Status:       s.Status,
				Priority:     s.Priority,
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}
			if s.MonthlyAllocation.IsPositive() {
				alloc := s.MonthlyAllocation.Mul(scale).Round(2)
				fundDay := 1
				goal.MonthlyAllocation = &alloc
				goal.FundDayOfMonth = &fundDay
			}
			if err := db.WithContext(ctx).Create(&goal).Error; err != nil {
				return err
			}

			contributions := []models.SavingContribution{{
				UserID:    u.ID,
				GoalID:    goal.ID,
				Amount:    s.InitialAmount.Mul(scale).Round(2),
				Month:     createdAt,
				Source:    models.SavingContributionSourceManual,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			}}
			if goal.MonthlyAllocation != nil {
				for m := s.CreatedMonthsAgo - 1; m >= 0; m-- {
					month := currentMonth.AddDate(0, -m, 0)
					contributions = append(contributions, models.SavingContribution{
						UserID:    u.ID,
						GoalID:    goal.ID,
						Amount:    *goal.MonthlyAllocation,
						Month:     month,
						Source:    models.SavingContributionSourceAuto,
						CreatedAt: month,
						UpdatedAt: month,
					})
				}
			}

			currentAmount := decimal.Zero
			for _, c := range contributions {
				if err := db.WithContext(ctx).Create(&c).Error; err != nil {
					return err
				}
				currentAmount = currentAmount.Add(c.Amount)
			}

			if err := db.WithContext(ctx).Model(&models.SavingGoal{}).
				Where("id = ?", goal.ID).
				Update("current_amount", currentAmount).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
