package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username      string         `gorm:"unique;not null" json:"username"`
	Password      string         `gorm:"not null" json:"-"` // do not output the password
	Email         string         `gorm:"unique;not null" json:"email"`
	DisplayName   string         `gorm:"not null" json:"display_name"`
	EmailVerified *time.Time     `json:"email_verified,omitempty"`
	RoleID        int64          `gorm:"not null" json:"role_id"`
	Role          Role           `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type Invitation struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	DisplayName string    `gorm:"not null" json:"display_name"`
	Username    string    `json:"username"`
	Email       string    `gorm:"type:varchar(255);index:idx_email_role,unique" json:"email"`
	Hash        string    `gorm:"type:varchar(255);not null" json:"hash"`
	RoleID      int64     `gorm:"not null;index:idx_email_role,unique" json:"role_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InvitationRequest struct {
	DisplayName string `json:"display_name" validate:"required"`
	Username    string `json:"username"`
	Email       string `json:"email" validate:"required,email"`
	Role        Role   `json:"role" validate:"required"`
}
