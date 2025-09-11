package workers

import (
	"context"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/rand"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
)

func SeedTransactions(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	var users []models.User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		return err
	}

	accRepo := repositories.NewAccountRepository(db)
	txnRepo := repositories.NewTransactionRepository(db)

	accService := services.NewAccountService(
		nil,
		nil,
		accRepo,
		txnRepo,
	)

	for _, u := range users {

		var accounts []models.Account
		if err := db.WithContext(ctx).
			Joins("JOIN account_types at ON at.id = accounts.account_type_id").
			Where("accounts.user_id = ? AND LOWER(at.sub_type) = ?", u.ID, "checking").
			Find(&accounts).Error; err != nil {
			return err
		}
		if len(accounts) == 0 {
			logger.Info("no checking accounts for user", zap.Int64("user_id", u.ID))
			continue
		}

		acc := accounts[0]

		// create ~200 random txns in that year
		for i := 0; i < 200; i++ {

			daysAgo := rng.Intn(365)
			date := today.AddDate(0, 0, -daysAgo)

			var ttype string
			if rng.Float64() < 0.4 {
				ttype = "income"
			} else {
				ttype = "expense"
			}

			amt := decimal.NewFromFloat(10 + rng.Float64()*1000).Round(2)

			desc := "Random " + ttype

			var category models.Category
			_ = db.Model(&models.Category{}).
				Where("classification = ?", "uncategorized").
				Order("name").
				First(&category)

			t := models.Transaction{
				UserID:          u.ID,
				AccountID:       acc.ID,
				TransactionType: ttype,
				CategoryID:      &category.ID,
				Amount:          amt,
				Currency:        acc.Currency,
				TxnDate:         date,
				Description:     &desc,
				IsAdjustment:    false,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			if err := db.WithContext(ctx).Create(&t).Error; err != nil {
				return err
			}

			if err := accService.UpdateAccountCashBalance(db, &acc, t.TxnDate, ttype, amt); err != nil {
				return err
			}

		}

		logger.Info("seeded transactions",
			zap.Int64("user_id", u.ID),
			zap.Int("count", 200),
			zap.String("account", acc.Name),
		)
	}

	return nil
}
