package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID                    uint               `gorm:"primaryKey;autoIncrement" json:"id"`
	Username              string             `gorm:"unique;not null" json:"username"`
	Password              string             `gorm:"not null" json:"-"` // do not output the password
	Email                 string             `gorm:"unique;not null" json:"email"`
	DisplayName           string             `gorm:"not null" json:"display_name"`
	EmailVerified         *time.Time         `json:"email_verified,omitempty"`
	RoleID                uint               `gorm:"not null" json:"role_id"`
	Role                  Role               `gorm:"foreignKey:RoleID" json:"role"`
	PrimaryOrganizationID *uint              `json:"primary_organization_id,omitempty"`
	PrimaryOrganization   Organization       `gorm:"foreignKey:PrimaryOrganizationID" json:"primary_organization,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	DeletedAt             gorm.DeletedAt     `gorm:"index" json:"-"`
	Secrets               *UserSecret        `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID" json:"secrets,omitempty"`
	Organizations         []OrganizationUser `gorm:"foreignKey:UserID" json:"organizations"`
}

type UserSecret struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint           `gorm:"not null;index:idx_user_id" json:"user_id"`
	BudgetInitialized bool           `gorm:"not null;default:false" json:"budget_initialized"`
	AllowLog          bool           `gorm:"default:true" json:"allow_log"`
	LastLogin         *time.Time     `json:"last_login,omitempty"`
	LastLoginIP       *string        `gorm:"index:idx_last_login_ip" json:"last_login_ip,omitempty"`
	BackupEmail       *string        `gorm:"size:100" json:"backup_email,omitempty"`
	TwoFactorSecret   *string        `json:"two_factor_secret,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
