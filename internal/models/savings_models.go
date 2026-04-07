package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type SavingGoalStatus string

const (
	SavingGoalStatusActive    SavingGoalStatus = "active"
	SavingGoalStatusPaused    SavingGoalStatus = "paused"
	SavingGoalStatusCompleted SavingGoalStatus = "completed"
	SavingGoalStatusArchived  SavingGoalStatus = "archived"
)

type SavingContributionSource string

const (
	SavingContributionSourceManual SavingContributionSource = "manual"
	SavingContributionSourceAuto   SavingContributionSource = "auto"
)

type SavingGoal struct {
	ID            int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64            `gorm:"not null" json:"user_id"`
	AccountID     int64            `gorm:"not null" json:"account_id"`
	Name          string           `gorm:"type:varchar(150);not null" json:"name"`
	TargetAmount  decimal.Decimal  `gorm:"type:decimal(19,4);not null" json:"target_amount"`
	CurrentAmount decimal.Decimal  `gorm:"type:decimal(19,4);not null;default:0" json:"current_amount"`
	TargetDate        *time.Time       `gorm:"type:date" json:"target_date,omitempty"`
	Status            SavingGoalStatus `gorm:"type:saving_goal_status;not null;default:active" json:"status"`
	Priority          int              `gorm:"not null;default:0" json:"priority"`
	MonthlyAllocation *decimal.Decimal `gorm:"type:decimal(19,4)" json:"monthly_allocation,omitempty"`
	FundDayOfMonth    *int             `gorm:"type:smallint"     json:"fund_day_of_month,omitempty"`
	CreatedAt     time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

type SavingContribution struct {
	ID        int64                    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64                    `gorm:"not null" json:"user_id"`
	GoalID    int64                    `gorm:"not null" json:"goal_id"`
	Amount    decimal.Decimal          `gorm:"type:decimal(19,4);not null" json:"amount"`
	Month     time.Time                `gorm:"type:date;not null" json:"month"`
	Note      *string                  `gorm:"type:varchar(255)" json:"note,omitempty"`
	Source    SavingContributionSource `gorm:"type:saving_contribution_source;not null;default:manual" json:"source"`
	CreatedAt time.Time                `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time                `gorm:"autoUpdateTime" json:"updated_at"`
}

type SavingGoalWithProgress struct {
	SavingGoal
	ProgressPercent decimal.Decimal  `json:"progress_percent"`
	TrackStatus     string           `json:"track_status"` // on_track, late, early, completed, no_target
	MonthsRemaining *int             `json:"months_remaining,omitempty"`
	MonthlyNeeded   *decimal.Decimal `json:"monthly_needed,omitempty"`
}

type SavingGoalReq struct {
	AccountID         int64            `json:"account_id" validate:"required"`
	Name              string           `json:"name" validate:"required,min=1,max=150"`
	TargetAmount      decimal.Decimal  `json:"target_amount" validate:"required"`
	InitialAmount     *decimal.Decimal `json:"initial_amount,omitempty"`
	TargetDate        *string          `json:"target_date,omitempty"`
	Priority          int              `json:"priority"`
	MonthlyAllocation *decimal.Decimal `json:"monthly_allocation,omitempty"`
	FundDayOfMonth    *int             `json:"fund_day_of_month,omitempty"`
}

type SavingGoalUpdateReq struct {
	Name              string           `json:"name" validate:"required,min=1,max=150"`
	TargetAmount      decimal.Decimal  `json:"target_amount" validate:"required"`
	TargetDate        *string          `json:"target_date,omitempty"`
	Status            SavingGoalStatus `json:"status" validate:"required"`
	Priority          int              `json:"priority"`
	MonthlyAllocation *decimal.Decimal `json:"monthly_allocation,omitempty"`
	FundDayOfMonth    *int             `json:"fund_day_of_month,omitempty"`
}

type SavingContributionReq struct {
	Amount decimal.Decimal `json:"amount" validate:"required"`
	Month  string          `json:"month" validate:"required"`
	Note   *string         `json:"note,omitempty"`
}
