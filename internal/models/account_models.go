package models

import "time"

type Account struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64    `gorm:"not null;index:idx_accounts_user" json:"user_id"`
	Name           string    `gorm:"type:varchar(150);not null" json:"name"`
	Subtype        *string   `gorm:"type:varchar(100)" json:"subtype,omitempty"` // e.g., checking, loan, credit_card
	Classification string    `gorm:"->;type:varchar(20)" json:"classification"`  // read-only
	Currency       string    `gorm:"type:char(4);not null;default:'EUR'" json:"currency"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
