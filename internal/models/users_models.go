package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username      string         `gorm:"unique;not null" json:"username"`
	Password      string         `gorm:"not null" json:"-"` // do not output the password
	Email         string         `gorm:"unique;not null" json:"email"`
	DisplayName   string         `gorm:"not null" json:"display_name"`
	EmailVerified *time.Time     `json:"email_verified,omitempty"`
	RoleID        uint           `gorm:"not null" json:"role_id"`
	Role          Role           `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
