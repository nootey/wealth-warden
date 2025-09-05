package models

import "time"

type SettingsGeneral struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	SupportEmail  *string   `gorm:"column:support_email" json:"support_email"`
	AllowSignups  bool      `gorm:"column:allow_signups" json:"allow_signups"`
	DefaultLocale string    `gorm:"column:default_locale" json:"default_locale"`
	DefaultTZ     string    `gorm:"column:default_timezone" json:"default_timezone"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (SettingsGeneral) TableName() string {
	return "settings_general"
}

type SettingsUser struct {
	UserID    int64     `gorm:"column:user_id;primaryKey" json:"id"`
	Theme     string    `gorm:"column:theme" json:"theme"`
	Accent    *string   `gorm:"column:accent" json:"accent"`
	Language  string    `gorm:"column:language" json:"language"`
	Timezone  string    `gorm:"column:timezone" json:"timezone"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (SettingsUser) TableName() string {
	return "settings_user"
}

type SettingsUserReq struct {
	Theme    string  `json:"theme"`
	Accent   *string `json:"accent"`
	Language string  `json:"language"`
	Timezone string  `json:"timezone"`
}
