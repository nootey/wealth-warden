package repositories

import (
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
)

type SettingsRepository struct {
	DB *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{DB: db}
}

func (r *SettingsRepository) FetchMaxAccountsForUser(tx *gorm.DB) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var maxAccounts int64
	err := db.Model(&models.SettingsGeneral{}).Select("max_accounts_per_user").First(&maxAccounts).Error
	if err != nil {
		return 0, err
	}

	return maxAccounts, nil
}

func (r *SettingsRepository) FetchGeneralSettings(tx *gorm.DB) (*models.SettingsGeneral, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var settings models.SettingsGeneral
	err := db.First(&settings).Error

	if err != nil {
		return nil, err
	}

	return &settings, err
}

func (r *SettingsRepository) FetchUserSettings(tx *gorm.DB, userID int64) (*models.SettingsUser, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var settings models.SettingsUser
	err := db.
		Where("user_id = ?", userID).
		First(&settings).
		Error

	if err != nil {
		return nil, err
	}

	return &settings, err
}

func (r *SettingsRepository) UpdateUserSettings(tx *gorm.DB, user *models.User, record models.SettingsUser) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	updates := map[string]any{
		"theme":      record.Theme,
		"accent":     record.Accent,
		"language":   record.Language,
		"timezone":   record.Timezone,
		"updated_at": time.Now(),
	}

	res := db.Model(&models.SettingsUser{}).
		Where("user_id = ?", user.ID).
		Updates(updates)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
