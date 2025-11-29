package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type SettingsRepositoryInterface interface {
	FetchMaxAccountsForUser(ctx context.Context, tx *gorm.DB) (int64, error)
	FetchGeneralSettings(ctx context.Context, tx *gorm.DB) (*models.SettingsGeneral, error)
	FetchUserSettings(ctx context.Context, tx *gorm.DB, userID int64) (*models.SettingsUser, error)
	UpdateUserSettings(ctx context.Context, tx *gorm.DB, userID int64, record models.SettingsUser) error
}

type SettingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) FetchMaxAccountsForUser(ctx context.Context, tx *gorm.DB) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var maxAccounts int64
	err := db.Model(&models.SettingsGeneral{}).Select("max_accounts_per_user").First(&maxAccounts).Error
	if err != nil {
		return 0, err
	}

	return maxAccounts, nil
}

func (r *SettingsRepository) FetchGeneralSettings(ctx context.Context, tx *gorm.DB) (*models.SettingsGeneral, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var settings models.SettingsGeneral
	err := db.First(&settings).Error

	if err != nil {
		return nil, err
	}

	return &settings, err
}

func (r *SettingsRepository) FetchUserSettings(ctx context.Context, tx *gorm.DB, userID int64) (*models.SettingsUser, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *SettingsRepository) UpdateUserSettings(ctx context.Context, tx *gorm.DB, userID int64, record models.SettingsUser) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	updates := map[string]any{
		"theme":      record.Theme,
		"accent":     record.Accent,
		"language":   record.Language,
		"timezone":   record.Timezone,
		"updated_at": time.Now().UTC(),
	}

	res := db.Model(&models.SettingsUser{}).
		Where("user_id = ?", userID).
		Updates(updates)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
