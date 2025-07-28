package models

import "time"

type Category struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64    `gorm:"not null;index:idx_categories_user_class" json:"user_id"`
	Name           string    `gorm:"type:varchar(100);not null" json:"name"`
	Classification string    `gorm:"type:enum('income','expense','savings','investment');not null;default:'expense';index:idx_categories_user_class" json:"classification"`
	ParentID       *uint64   `gorm:"index" json:"parent_id,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
