package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID            uint       `gorm:"primaryKey;autoIncrement"`
	Username      string     `gorm:"unique;not null"`
	Password      string     `gorm:"not null"`
	Email         string     `gorm:"not null;unique;not null"`
	DisplayName   *string    `gorm:"default:null"`
	Role          string     `gorm:"default:member;not null"`
	EmailVerified *time.Time `gorm:"default:null"`
	AllowLog      bool       `gorm:"default:true"`
	LastLogin     *time.Time `gorm:"default:null"`
	LastLoginIP   *string    `gorm:"default:null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
