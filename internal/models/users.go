package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID            uint       `gorm:"primaryKey;autoIncrement"`
	Username      string     `gorm:"not null"`
	Email         string     `gorm:"not null;unique"`
	DisplayName   *string    `gorm:"default:null"`
	EmailVerified *time.Time `gorm:"default:null"`
	LastLogin     *time.Time `gorm:"default:null"`
	LastLoginIP   *string    `gorm:"default:null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
