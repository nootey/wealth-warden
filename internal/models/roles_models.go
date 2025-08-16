package models

import "time"

type Role struct {
	ID              int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string           `gorm:"unique;not null;size:75" json:"name"`
	IsGlobal        bool             `gorm:"not null" json:"is_global"`
	Description     string           `gorm:"size:255" json:"description"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Users           []User           `gorm:"foreignKey:RoleID" json:"-"`
	RolePermissions []RolePermission `gorm:"foreignKey:RoleID" json:"-"`
}

type Permission struct {
	ID              int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string           `gorm:"unique;not null;size:75" json:"name"`
	Description     string           `gorm:"size:255" json:"description"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	RolePermissions []RolePermission `gorm:"foreignKey:PermissionID" json:"-"`
}

type RolePermission struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID       int64      `gorm:"not null;uniqueIndex:idx_role_permission" json:"role_id"`
	PermissionID int64      `gorm:"not null;uniqueIndex:idx_role_permission" json:"permission_id"`
	Role         Role       `gorm:"constraint:OnDelete:CASCADE;foreignKey:RoleID" json:"-"`
	Permission   Permission `gorm:"constraint:OnDelete:CASCADE;foreignKey:PermissionID" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
