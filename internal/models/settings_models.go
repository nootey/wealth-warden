package models

import "time"

type SettingsGeneral struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	SupportEmail    *string   `gorm:"column:support_email" json:"support_email"`
	AllowSignups    bool      `gorm:"column:allow_signups" json:"allow_signups"`
	DefaultLocale   string    `gorm:"column:default_locale" json:"default_locale"`
	DefaultTZ       string    `gorm:"column:default_timezone" json:"default_timezone"`
	MaxUserAccounts int       `gorm:"column:max_accounts_per_user" json:"max_user_accounts"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
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

type TimezoneInfo struct {
	Value       string `json:"value"`       // e.g., "America/New_York"
	Label       string `json:"label"`       // e.g., "(UTC-05:00) America/New_York"
	Offset      int    `json:"offset"`      // Offset in seconds
	DisplayName string `json:"displayName"` // Human-readable name
}

type PreferenceSettingsReq struct {
	Theme    string  `json:"theme"`
	Accent   *string `json:"accent"`
	Language string  `json:"language"`
	Timezone string  `json:"timezone"`
}

type ProfileSettingsReq struct {
	DisplayName  string `json:"display_name" validate:"required"`
	Email        string `json:"email" validate:"required"`
	EmailUpdated bool   `json:"email_updated"`
}

type BackupInfo struct {
	Name     string         `json:"name"`
	Metadata BackupMetadata `json:"metadata"`
}

type BackupMetadata struct {
	AppVersion string    `json:"app_version"`
	CommitSHA  string    `json:"commit_sha"`
	BuildTime  string    `json:"build_time"`
	DBVersion  int64     `json:"db_version"`
	CreatedAt  time.Time `json:"created_at"`
	BackupSize int64     `json:"backup_size"` // in bytes
}
