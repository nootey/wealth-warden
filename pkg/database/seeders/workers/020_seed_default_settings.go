package workers

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func strPtr(s string) *string {
	return &s
}

func SeedDefaultSettings(ctx context.Context, db *gorm.DB, logger *zap.Logger, cfg *config.Config) error {
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
		logger.Error("failed to seed settings_general", zap.Error(err))
		return err
	}

	logger.Info("seeded settings_general successfully")
	return nil
}
