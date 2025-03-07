package models

import (
	"gorm.io/gorm"
	"time"
)

type Organization struct {
	ID               uint               `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string             `gorm:"not null" json:"name"`
	OrganizationType string             `gorm:"default:solo" json:"organization_type"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Users            []OrganizationUser `gorm:"foreignKey:OrganizationID" json:"-"`
}

type OrganizationUser struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint           `gorm:"not null;uniqueIndex:idx_user_organization" json:"user_id"`
	OrganizationID uint           `gorm:"not null;uniqueIndex:idx_user_organization" json:"organization_id"`
	RoleID         uint           `gorm:"not null" json:"role_id"`
	User           User           `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID" json:"-"`
	Organization   Organization   `gorm:"constraint:OnDelete:CASCADE;foreignKey:OrganizationID" json:"organization"`
	Role           Role           `gorm:"constraint:OnDelete:CASCADE;foreignKey:RoleID" json:"role"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
