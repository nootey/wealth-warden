package workers

import (
	"context"
	"fmt"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func strPtr(s string) *string {
	return &s
}

func SeedDefaultSettings(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	defaults := models.SettingsGeneral{
		SupportEmail:    strPtr("support@wealth.warden"),
		AllowSignups:    true,
		DefaultLocale:   "en",
		DefaultTZ:       "UTC",
		MaxUserAccounts: 25,
	}

	if err := db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&defaults).Error; err != nil {
		return fmt.Errorf("failed to seed settings_general %w", err)
	}

	return nil
}
