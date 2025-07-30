package models

import "time"

type Account struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint      `gorm:"not null;index:idx_accounts_user" json:"user_id"`
	Name          string    `gorm:"type:varchar(150);not null" json:"name"`
	AccountTypeID uint      `gorm:"not null" json:"account_type_id"`
	Currency      string    `gorm:"type:char(4);not null;default:'EUR'" json:"currency"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type AccountType struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type           string    `gorm:"type:varchar(150);not null" json:"type"`
	Subtype        *string   `gorm:"type:varchar(100)" json:"subtype"`
	Classification string    `gorm:"->;type:varchar(20)" json:"classification"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
