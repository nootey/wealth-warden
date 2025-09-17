package models

import "time"

type Role struct {
	ID          int64        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string       `gorm:"unique;not null;size:75" json:"name"`
	IsDefault   bool         `gorm:"not null" json:"is_default"`
	Description *string      `gorm:"size:255" json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions"`
}

type Permission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"unique;not null;size:75" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoleReq struct {
	Name        string       `json:"name" validate:"required"`
	IsDefault   bool         `json:"is_default"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}
