package models

import "time"

type SavingsSummary struct {
	Month        int      `json:"month"`
	CategoryID   uint     `json:"category_id" gorm:"column:category_id"`
	CategoryName string   `json:"category_name" gorm:"column:category_name"`
	CategoryType string   `json:"category_type" gorm:"column:category_type"`
	TotalAmount  float64  `json:"total_amount" gorm:"column:total_amount"`
	GoalProgress *float64 `json:"goal_progress"`
	GoalTarget   *float64 `json:"goal_target"`
	GoalSpent    *float64 `json:"goal_spent"`
}

type SavingsCategory struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	OrganizationID  uint      `gorm:"not_null;index" json:"organization_id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	SavingsType     string    `gorm:"type:enum('fixed', 'variable');not null" json:"savings_type"`
	Priority        int       `gorm:"default:1" json:"priority"`
	GoalTarget      *float64  `gorm:"type:decimal(10,2)" json:"goal_target,omitempty"`
	GoalProgress    float64   `gorm:"type:decimal(10,2);default:0.00" json:"goal_progress"`
	AccountType     string    `gorm:"type:varchar(128);default:'normal'" json:"account_type"` // normal, interest
	InterestRate    *float64  `gorm:"type:decimal(5,2)" json:"interest_rate,omitempty"`
	AccruedInterest float64   `gorm:"type:decimal(10,2);default:0.00" json:"accrued_interest"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SavingsTransaction struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	OrganizationID    uint            `gorm:"not_null;index" json:"organization_id"`
	UserID            uint            `gorm:"not_null" json:"user_id"`
	SavingsCategoryID uint            `gorm:"index" json:"savings_category_id"`
	SavingsCategory   SavingsCategory `gorm:"foreignKey:SavingsCategoryID" json:"savings_category"`
	TransactionType   string          `gorm:"type:varchar(24);not null;enum('allocation','deduction')" json:"transaction_type"`
	TransactionDate   time.Time       `gorm:"not null" json:"transaction_date"`
	AllocatedAmount   float64         `gorm:"type:decimal(10,2)" json:"allocated_amount"`
	AdjustedAmount    float64         `gorm:"type:decimal(10,2)" json:"adjusted_amount"`
	Description       *string         `gorm:"type:varchar(255);not null" json:"description,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
