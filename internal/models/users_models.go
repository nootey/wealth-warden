package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Password       string         `gorm:"not null" json:"-"` // do not output the password
	Email          string         `gorm:"unique;not null" json:"email"`
	DisplayName    string         `gorm:"not null" json:"display_name"`
	EmailConfirmed *time.Time     `json:"email_confirmed"`
	RoleID         int64          `gorm:"not null" json:"role_id"`
	Role           Role           `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type Invitation struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	DisplayName string    `gorm:"not null" json:"display_name"`
	Email       string    `gorm:"type:varchar(255);index:idx_email_role,unique" json:"email"`
	Hash        string    `gorm:"type:varchar(255);not null" json:"hash"`
	RoleID      int64     `gorm:"not null;index:idx_email_role,unique" json:"role_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Token struct {
	ID         int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenType  string            `gorm:"not null" json:"token_type"`
	TokenValue string            `json:"token_value"`
	Data       datatypes.JSONMap `gorm:"type:jsonb" json:"data"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type InvitationReq struct {
	DisplayName string `json:"display_name" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	RoleID      int64  `json:"role_id" validate:"required"`
}

type UserReq struct {
	Email                string  `json:"email" validate:"required,email"`
	DisplayName          string  `json:"display_name" validate:"required"`
	RoleID               int64   `json:"role_id" validate:"required"`
	Password             *string `json:"password"`
	PasswordConfirmation *string `json:"password_confirmation"`
}
