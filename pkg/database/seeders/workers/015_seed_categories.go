package workers

import (
	"context"
	"fmt"
	"wealth-warden/internal/config"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SeedCategories gives every seeded demo user their own copy of the preset default
// categories. Categories are per-user, so there's no longer a single shared set to seed.
func SeedCategories(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	var users []models.User
	if err := db.WithContext(ctx).Where("display_name IN ?", seededUsernames).Find(&users).Error; err != nil {
		return err
	}

	txnRepo := repositories.NewTransactionRepository(db)
	accRepo := repositories.NewAccountRepository(db)
	balanceRepo := repositories.NewBalanceRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)
	txnService := services.NewTransactionService(zap.NewNop(), txnRepo, accRepo, balanceRepo, settingsRepo, savingsRepo, jobqueue.NoopDispatcher{})

	for _, u := range users {
		if _, err := txnService.SeedDefaultCategoriesWithTx(ctx, db, u.ID); err != nil {
			return fmt.Errorf("failed to seed default categories for user %d: %w", u.ID, err)
		}
	}
	return nil
}
